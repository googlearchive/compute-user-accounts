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

// These are mocked in tests.
var (
	utcTime    = func() time.Time { return time.Now().UTC() }
	pulseAfter = time.After
	// accountRefreshCallback exposes a testing callback invoked when the
	// store has refreshed user and group data.
	accountRefreshCallback = func() {}
	// keyRefreshCallback exposes a testing callback invoked when the store
	// has refreshed all key data.
	keyRefreshCallback = func() {}
)

// A Config provides configuration options for a store AccountProvider.
type Config struct {
	// AccountRefreshFrequency defines how often to perform scheduled
	// refreshes of user and group information.
	AccountRefreshFrequency time.Duration
	// AccountRefreshCooldown defines how long to block on-demand refreshes
	// of user and group information after a refresh.
	AccountRefreshCooldown time.Duration
	// KeyRefreshFrequency defines how often to perform scheduled refreshes
	// of authorized keys information.
	KeyRefreshFrequency time.Duration
	// KeyRefreshCooldown defines how long to block on-demand refreshes of
	// authorized keys information after a refresh.
	KeyRefreshCooldown time.Duration
}

type cachedUser struct {
	user           *accounts.User
	keys           []string
	keyRefreshTime time.Time
}

// cachingStore implements AccountProvider as an in-memory store.
type cachingStore struct {
	sync.RWMutex
	apiClient     apiclient.APIClient
	config        *Config
	updateWaiters chan chan struct{}
	usersByName   map[string]*cachedUser
	usersByUID    map[uint32]*cachedUser
	groupsByName  map[string]*accounts.Group
	groupsByGID   map[uint32]*accounts.Group
}

// New returns an AccountProvider implemented as an in-memory store.
func New(apiClient apiclient.APIClient, config *Config) accounts.AccountProvider {
	store := &cachingStore{
		apiClient:     apiClient,
		config:        config,
		updateWaiters: make(chan chan struct{}),
	}
	ch := make(chan struct{})
	go updateTask(store)
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
		case <-pulseAfter(s.config.AccountRefreshFrequency):
		}
		if nowOutsideTimespan(lastRefresh, s.config.AccountRefreshCooldown) {
			logger.Info("Refreshing users and groups.")
			updateAccounts(s)
			lastRefresh = utcTime()
		}
		accountRefreshCallback()
		if ch != nil {
			close(ch)
		}
	}
}

func updateAccounts(s *cachingStore) {
	defer func() { go updateKeys(s) }()
	users, groups, err := s.apiClient.UsersAndGroups()
	if err != nil {
		logger.Errorf("Failed refresh: %v.", err)
		return
	}
	s.Lock()
	defer s.Unlock()
	oldUsers := s.usersByName
	s.usersByName = make(map[string]*cachedUser)
	s.usersByUID = make(map[uint32]*cachedUser)
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
		cu := &cachedUser{user: user}
		if old, ok := oldUsers[user.Name]; ok {
			cu.keyRefreshTime = old.keyRefreshTime
			cu.keys = old.keys
		}
		s.usersByName[user.Name] = cu
		s.usersByUID[user.UID] = cu
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
	logger.Info("Refreshing users and groups succeeded.")
}

func updateKeys(s *cachingStore) {
	type update struct {
		name string
		keys []string
		err  error
		time time.Time
	}
	ch := make(chan update, 16)
	workers := 0
	for _, name := range keysRequiringRefresh(s) {
		go func(n string) {
			keys, err := s.apiClient.AuthorizedKeys(n)
			ch <- update{n, keys, err, utcTime()}
		}(name)
		workers += 1
	}
	refreshedKeys := make([]update, workers)
	for workers != 0 {
		up := <-ch
		workers -= 1
		if up.err != nil {
			logger.Errorf("Failed key refresh for %v: %v.", up.name, up.err)
			continue
		}
		logger.Infof("Refreshed keys for %v.", up.name)
		refreshedKeys = append(refreshedKeys, up)
	}
	s.Lock()
	defer s.Unlock()
	for _, rk := range refreshedKeys {
		if cu, ok := s.usersByName[rk.name]; ok {
			cu.keys = rk.keys
			cu.keyRefreshTime = rk.time
		}
	}
	keyRefreshCallback()
}

func keysRequiringRefresh(s *cachingStore) []string {
	var result []string
	s.RLock()
	defer s.RUnlock()
	for name, cu := range s.usersByName {
		if nowOutsideTimespan(cu.keyRefreshTime, s.config.KeyRefreshFrequency) {
			result = append(result, name)
		}
	}
	return result
}

func (s *cachingStore) userByNameImpl(name string) (*cachedUser, bool) {
	s.RLock()
	defer s.RUnlock()
	cu, ok := s.usersByName[name]
	return cu, ok
}

// UserByName satisfies AccountProvider.
func (s *cachingStore) UserByName(name string) (*accounts.User, error) {
	cu, ok := s.userByNameImpl(name)
	if ok {
		return cu.user, nil
	}
	ch := make(chan struct{})
	logger.Infof("Triggering refresh due to missing user %v.", name)
	s.updateWaiters <- ch
	// Block on update.
	<-ch
	cu, ok = s.userByNameImpl(name)
	if ok {
		return cu.user, nil
	}
	return nil, accounts.UsernameNotFound(name)
}

// UserByName satisfies AccountProvider.
func (s *cachingStore) UserByUID(uid uint32) (*accounts.User, error) {
	s.RLock()
	defer s.RUnlock()
	cu, ok := s.usersByUID[uid]
	if ok {
		return cu.user, nil
	}
	return nil, accounts.UIDNotFound(uid)
}

// Users satisfies AccountProvider.
func (s *cachingStore) Users() ([]*accounts.User, error) {
	s.RLock()
	defer s.RUnlock()
	ret := make([]*accounts.User, len(s.usersByName))
	i := 0
	for _, cu := range s.usersByName {
		ret[i] = cu.user
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
	cu, ok := s.userByNameImpl(username)
	if !ok {
		return nil, accounts.UsernameNotFound(username)
	} else if !nowOutsideTimespan(cu.keyRefreshTime, s.config.KeyRefreshCooldown) {
		logger.Infof("Returning cached keys for %v due to cooldown.", username)
		return cu.keys, nil
	}
	keys, err := s.apiClient.AuthorizedKeys(username)
	if err != nil {
		return cu.keys, nil
	}
	go s.updateCachedKeys(username, keys, utcTime())
	return keys, nil
}

func (s *cachingStore) updateCachedKeys(username string, keys []string, refreshTime time.Time) {
	s.Lock()
	defer s.Unlock()
	if cu, ok := s.usersByName[username]; ok {
		cu.keys = keys
		cu.keyRefreshTime = refreshTime
	}
}
