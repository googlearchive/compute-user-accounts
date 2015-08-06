// Copyright 2015 Google Inc. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package store provides an in-memory store of account data.
package store

import (
	"sync"
	"time"

	"github.com/GoogleCloudPlatform/compute-user-accounts/accounts"
	"github.com/GoogleCloudPlatform/compute-user-accounts/apiclient"
	"github.com/GoogleCloudPlatform/compute-user-accounts/logger"
)

// utcTime allows dependency injection for testing.
var utcTime = func() time.Time { return time.Now().UTC() }

// pulseAfter allows dependency injection for testing.
var pulseAfter = func(d time.Duration) <-chan time.Time { return time.After(d) }

// refreshCallback exposes a testing callback invoked when the store has
// refreshed user and group data.
var refreshCallback = func() {}

// A Config provides configuration options for a store AccountProvider.
type Config struct {
	// RefreshFrequency defines how often to perform scheduled refreshes of
	// user and group information.
	RefreshFrequency time.Duration
	// RefreshCooldown defines how long to block on-demand refreshes after a
	// refresh.
	RefreshCooldown time.Duration
	// KeyExpiration defines how long to cache authorized keys in case of
	// inability to communicate with the Compute Accounts API.
	KeyExpiration time.Duration
	// KeyCooldown defines how long to serve authorized keys from the cache
	// after they are fetched.
	KeyCooldown time.Duration
}

type cachedKeys struct {
	values       []string
	creationTime time.Time
}

// cachingStore implements AccountProvider as an in-memory store.
type cachingStore struct {
	sync.RWMutex
	apiClient      apiclient.APIClient
	config         *Config
	updateWaiters  chan chan struct{}
	usersByName    map[string]*accounts.User
	usersByUID     map[uint32]*accounts.User
	groupsByName   map[string]*accounts.Group
	groupsByGID    map[uint32]*accounts.Group
	keysByUsername map[string]*cachedKeys
}

// New returns an AccountProvider implemented as an in-memory store.
func New(apiClient apiclient.APIClient, config *Config) accounts.AccountProvider {
	store := &cachingStore{
		apiClient:      apiClient,
		config:         config,
		updateWaiters:  make(chan chan struct{}),
		keysByUsername: make(map[string]*cachedKeys),
	}
	go updateTask(store)
	ch := make(chan struct{})
	store.updateWaiters <- ch
	<-ch
	return store
}

func nowOutsideTimespan(start time.Time, duration time.Duration) bool {
	now := utcTime()
	end := start.Add(duration)
	return now.Before(start) || now.After(end)
}

func updateTask(s *cachingStore) {
	var lastRefresh time.Time
	for {
		var ch chan struct{}
		select {
		case ch = <-s.updateWaiters:
		case <-pulseAfter(s.config.RefreshFrequency):
		}
		if nowOutsideTimespan(lastRefresh, s.config.RefreshCooldown) {
			logger.Info("Refreshing users and groups.")
			updateCache(s)
			lastRefresh = utcTime()
		}
		refreshCallback()
		if ch != nil {
			close(ch)
		}
	}
}

func updateCache(s *cachingStore) {
	users, groups, err := s.apiClient.UsersAndGroups()
	if err != nil {
		logger.Errorf("Failed refresh: %v.", err)
		return
	}
	s.Lock()
	defer s.Unlock()
	s.usersByName = make(map[string]*accounts.User)
	s.usersByUID = make(map[uint32]*accounts.User)
	s.groupsByName = make(map[string]*accounts.Group)
	s.groupsByGID = make(map[uint32]*accounts.Group)
	for _, u := range users {
		user := &accounts.User{
			Name:          u.Username,
			UID:           uint32(u.Uid),
			GID:           uint32(u.Gid),
			Gecos:         u.Gecos,
			HomeDirectory: u.HomeDirectory,
			Shell:         u.Shell,
		}
		s.usersByName[user.Name] = user
		s.usersByUID[user.UID] = user
	}
	for _, g := range groups {
		group := &accounts.Group{
			Name:    g.GroupName,
			GID:     uint32(g.Gid),
			Members: g.Members,
		}
		s.groupsByName[group.Name] = group
		s.groupsByGID[group.GID] = group
	}
	for u, k := range s.keysByUsername {
		_, ok := s.keysByUsername[u]
		if !ok || s.areKeysExpired(k) {
			delete(s.keysByUsername, u)
		}
	}
	logger.Info("Refreshing succeeded.")
}

func (s *cachingStore) areKeysExpired(keys *cachedKeys) bool {
	return nowOutsideTimespan(keys.creationTime, s.config.KeyExpiration)
}

func (s *cachingStore) userByNameImpl(name string) (*accounts.User, bool) {
	s.RLock()
	defer s.RUnlock()
	u, ok := s.usersByName[name]
	return u, ok
}

// UserByName satisfies AccountProvider.
func (s *cachingStore) UserByName(name string) (*accounts.User, error) {
	u, ok := s.userByNameImpl(name)
	if ok {
		return u, nil
	}
	ch := make(chan struct{})
	logger.Info("Triggering refresh due to missing user.")
	s.updateWaiters <- ch
	// Block on update.
	<-ch
	u, ok = s.userByNameImpl(name)
	if ok {
		return u, nil
	}
	return nil, accounts.UsernameNotFound(name)
}

// UserByName satisfies AccountProvider.
func (s *cachingStore) UserByUID(uid uint32) (*accounts.User, error) {
	s.RLock()
	defer s.RUnlock()
	u, ok := s.usersByUID[uid]
	if ok {
		return u, nil
	}
	return nil, accounts.UIDNotFound(uid)
}

// Users satisfies AccountProvider.
func (s *cachingStore) Users() ([]*accounts.User, error) {
	s.RLock()
	defer s.RUnlock()
	ret := make([]*accounts.User, len(s.usersByName))
	i := 0
	for _, u := range s.usersByName {
		ret[i] = u
		i++
	}
	return ret, nil
}

// GroupByName satisfies AccountProvider.
func (s *cachingStore) GroupByName(name string) (*accounts.Group, error) {
	s.RLock()
	defer s.RUnlock()
	g, ok := s.groupsByName[name]
	if ok {
		return g, nil
	}
	ch := make(chan struct{})
	logger.Info("Triggering refresh due to missing group.")
	go func() { s.updateWaiters <- ch }()
	// Do not block on update.
	return nil, accounts.GroupNameNotFound(name)
}

// GroupByGID satisfies AccountProvider.
func (s *cachingStore) GroupByGID(gid uint32) (*accounts.Group, error) {
	s.RLock()
	defer s.RUnlock()
	g, ok := s.groupsByGID[gid]
	if ok {
		return g, nil
	}
	return nil, accounts.GIDNotFound(gid)
}

// Groups satisfies AccountProvider.
func (s *cachingStore) Groups() ([]*accounts.Group, error) {
	s.RLock()
	defer s.RUnlock()
	ret := make([]*accounts.Group, len(s.groupsByName))
	i := 0
	for _, g := range s.groupsByName {
		ret[i] = g
		i++
	}
	return ret, nil
}

// Names satisfies AccountProvider.
func (s *cachingStore) Names() ([]string, error) {
	s.RLock()
	defer s.RUnlock()
	ret := make([]string, len(s.usersByName)+len(s.groupsByName))
	i := 0
	for u := range s.usersByName {
		ret[i] = u
		i++
	}
	for g := range s.groupsByName {
		ret[i] = g
		i++
	}
	return ret, nil
}

// IsName satisfies AccountProvider.
func (s *cachingStore) IsName(name string) (bool, error) {
	s.RLock()
	defer s.RUnlock()
	_, ok1 := s.usersByName[name]
	_, ok2 := s.groupsByName[name]
	return ok1 || ok2, nil
}

// AuthorizedKeys satisfies AccountProvider.
func (s *cachingStore) AuthorizedKeys(username string) ([]string, error) {
	_, ok := s.userByNameImpl(username)
	if !ok {
		return nil, accounts.UsernameNotFound(username)
	}
	cacheEntry := s.cachedKeys(username)
	var cachedKeys []string
	if cacheEntry == nil {
		cachedKeys = nil
	} else if !nowOutsideTimespan(cacheEntry.creationTime, s.config.KeyCooldown) {
		logger.Infof("Returning cached keys due to cooldown.")
		return cacheEntry.values, nil
	} else {
		cachedKeys = cacheEntry.values
	}
	// If the API returns 404 this will return a nil slice, not an error.
	keys, err := s.apiClient.AuthorizedKeys(username)
	if err == nil {
		s.updateCachedKeys(username, keys)
		return keys, nil
	}
	if cachedKeys != nil {
		logger.Noticef("Returning cached keys due to failure: %v.", err)
		return cachedKeys, nil
	}
	return nil, err
}

func (s *cachingStore) updateCachedKeys(username string, keys []string) {
	s.Lock()
	defer s.Unlock()
	s.keysByUsername[username] = &cachedKeys{keys, utcTime()}
}

func (s *cachingStore) cachedKeys(username string) *cachedKeys {
	s.RLock()
	defer s.RUnlock()
	keys, ok := s.keysByUsername[username]
	if ok && !s.areKeysExpired(keys) {
		return keys
	}
	return nil
}
