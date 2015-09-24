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

package store

import (
	"errors"
	"sort"
	"testing"
	"time"

	"github.com/GoogleCloudPlatform/compute-user-accounts/accounts"
	"github.com/GoogleCloudPlatform/compute-user-accounts/testbase"

	_ "net/http/pprof"

	cua "google.golang.org/api/clouduseraccounts/vm_beta"
)

type mockAPIClient struct {
	users            []*cua.LinuxUserView
	groups           []*cua.LinuxGroupView
	usersGroupsError error
	keys             map[string][]string
	sudoers          map[string]bool
	keysError        error
}

// UsersAndGroups satisfies APIClient.
func (c *mockAPIClient) UsersAndGroups() ([]*cua.LinuxUserView, []*cua.LinuxGroupView, error) {
	return c.users, c.groups, c.usersGroupsError
}

// UsersAndGroups satisfies APIClient.
func (c *mockAPIClient) AuthorizedKeys(username string) (*cua.AuthorizedKeysView, error) {
	keys, ok1 := c.keys[username]
	sudoer, ok2 := c.sudoers[username]
	if !ok1 || !ok2 {
		// This case is equivalent to an API 404, return the nil slice.
		return nil, errors.New("invalid user")
	}
	return &cua.AuthorizedKeysView{
		Keys:   keys,
		Sudoer: sudoer,
	}, nil
}

type userSlice []*accounts.User

func (s userSlice) Len() int           { return len(s) }
func (s userSlice) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }
func (s userSlice) Less(i, j int) bool { return s[i].Name < s[j].Name }

type groupSlice []*accounts.Group

func (s groupSlice) Len() int           { return len(s) }
func (s groupSlice) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }
func (s groupSlice) Less(i, j int) bool { return s[i].Name < s[j].Name }

var expSudoers = &accounts.Group{
	Name:    "gce-sudoers",
	GID:     4001,
	Members: []string{"user1"},
}

func newMock() *mockAPIClient {
	return &mockAPIClient{
		users: []*cua.LinuxUserView{
			&cua.LinuxUserView{
				Username:      "user1",
				Uid:           1001,
				Gid:           1000,
				Gecos:         "John Doe",
				HomeDirectory: "/home/user1",
				Shell:         "/bin/bash",
			},
			&cua.LinuxUserView{
				Username:      "user2",
				Uid:           1002,
				Gid:           1000,
				Gecos:         "Jane Doe",
				HomeDirectory: "/home/user2",
				Shell:         "/bin/zsh",
			},
		},
		groups: []*cua.LinuxGroupView{
			&cua.LinuxGroupView{
				GroupName: "group1",
				Gid:       1000,
				Members:   []string(nil),
			},
			&cua.LinuxGroupView{
				GroupName: "group2",
				Gid:       1001,
				Members:   []string{"user2", "user1"},
			},
		},
		keys:    testbase.ExpKeys,
		sudoers: map[string]bool{"user1": true, "user3": false},
	}
}

func registerCallback(config *Config) chan struct{} {
	ch := make(chan struct{})
	refreshCallback = func(c *Config) {
		if c == config {
			close(ch)
		}
	}
	return ch
}

func removeCallback() {
	refreshCallback = func(*Config) {}
}

func testStore(mock *mockAPIClient, config *Config) accounts.AccountProvider {
	// Ensure keys are warmed.
	ch := registerCallback(config)
	store := New(mock, config)
	<-ch
	removeCallback()
	return store
}

func TestUsersGroups(t *testing.T) {
	mock := newMock()
	config := &Config{
		AccountRefreshFrequency: time.Hour,
		AccountRefreshCooldown:  0,
		KeyRefreshFrequency:     time.Hour,
		KeyRefreshCooldown:      0,
	}
	store := testStore(mock, config)
	testbase.RunCases(t, []testbase.TestCase{
		&testbase.SuccessCase{
			`UserByName("user1")`,
			func() (interface{}, error) { return store.UserByName("user1") },
			testbase.ExpUsers[0],
		},
		&testbase.SuccessCase{
			"UserByUID(1002)",
			func() (interface{}, error) { return store.UserByUID(1002) },
			testbase.ExpUsers[1],
		},
		&testbase.SuccessCase{
			`GroupByName("group1")`,
			func() (interface{}, error) { return store.GroupByName("group1") },
			testbase.ExpGroups[0],
		},
		&testbase.SuccessCase{
			`GroupByName("gce-sudoers")`,
			func() (interface{}, error) { return store.GroupByName("gce-sudoers") },
			expSudoers,
		},
		&testbase.SuccessCase{
			"GroupByGID(1001)",
			func() (interface{}, error) { return store.GroupByGID(1001) },
			testbase.ExpGroups[1],
		},
		&testbase.SuccessCase{
			"GroupByGID(4001)",
			func() (interface{}, error) { return store.GroupByGID(4001) },
			expSudoers,
		},
		&testbase.SuccessCase{
			"Users()",
			func() (interface{}, error) { r, e := store.Users(); sort.Sort(userSlice(r)); return r, e },
			testbase.ExpUsers,
		},
		&testbase.SuccessCase{
			"Groups()",
			func() (interface{}, error) { r, e := store.Groups(); sort.Sort(groupSlice(r)); return r, e },
			append([]*accounts.Group{expSudoers}, testbase.ExpGroups...),
		},
		&testbase.SuccessCase{
			"Names()",
			func() (interface{}, error) { r, e := store.Names(); sort.Sort(sort.StringSlice(r)); return r, e },
			append([]string{"gce-sudoers"}, testbase.ExpNames...),
		},
		&testbase.SuccessCase{
			`IsName("user1")`,
			func() (interface{}, error) { return store.IsName("user1") },
			true,
		},
		&testbase.SuccessCase{
			`IsName("group1")`,
			func() (interface{}, error) { return store.IsName("group1") },
			true,
		},
		&testbase.SuccessCase{
			`IsName("gce-sudoers")`,
			func() (interface{}, error) { return store.IsName("gce-sudoers") },
			true,
		},
		&testbase.SuccessCase{
			`IsName("nil")`,
			func() (interface{}, error) { return store.IsName("nil") },
			false,
		},
		&testbase.FailureCase{
			`UserByName("nil")`,
			func() (interface{}, error) { return store.UserByName("nil") },
			`unable to find user with name "nil"`,
		},
		&testbase.FailureCase{
			"UserByUID(2)",
			func() (interface{}, error) { return store.UserByUID(2) },
			"unable to find user with UID 2",
		},
		&testbase.FailureCase{
			`GroupByName("nil")`,
			func() (interface{}, error) { return store.GroupByName("nil") },
			`unable to find group with name "nil"`,
		},
		&testbase.FailureCase{
			"GroupByGID(1)",
			func() (interface{}, error) { return store.GroupByGID(1) },
			"unable to find group with GID 1",
		},
	})
}

func TestKeysBasicCase(t *testing.T) {
	mock := newMock()
	config := &Config{
		AccountRefreshFrequency: time.Hour,
		AccountRefreshCooldown:  0,
		KeyRefreshFrequency:     time.Hour,
		KeyRefreshCooldown:      0,
	}
	store := testStore(mock, config)
	testbase.RunCases(t, []testbase.TestCase{
		&testbase.SuccessCase{
			`AuthorizedKeys("user1")`,
			func() (interface{}, error) { return store.AuthorizedKeys("user1") },
			testbase.ExpKeys["user1"],
		},
		&testbase.SuccessCase{
			`AuthorizedKeys("user2")`,
			func() (interface{}, error) { return store.AuthorizedKeys("user2") },
			[]string(nil),
		},
		&testbase.FailureCase{
			`AuthorizedKeys("user3")`,
			func() (interface{}, error) { return store.AuthorizedKeys("user3") },
			`unable to find user with name "user3"`,
		},
	})
}

func TestKeyPrewarmingAndCaching(t *testing.T) {
	mock := newMock()
	// Background key refreshes happen every second.
	config := &Config{
		AccountRefreshFrequency: time.Hour,
		AccountRefreshCooldown:  0,
		KeyRefreshFrequency:     time.Hour,
		KeyRefreshCooldown:      0,
	}
	store := testStore(mock, config)
	mock.keysError = errors.New("API error")
	testbase.RunCases(t, []testbase.TestCase{
		&testbase.SuccessCase{
			`AuthorizedKeys("user1")`,
			func() (interface{}, error) { return store.AuthorizedKeys("user1") },
			testbase.ExpKeys["user1"],
		},
		&testbase.SuccessCase{
			`AuthorizedKeys("user2")`,
			func() (interface{}, error) { return store.AuthorizedKeys("user2") },
			[]string(nil),
		},
	})
}

func TestKeyCooldownAndRefresh(t *testing.T) {
	mTime := time.Now().UTC()
	// Mock time.
	timeNow = func() time.Time { return mTime }
	pulse := make(chan time.Time)
	timeAfter = func(time.Duration) <-chan time.Time { return pulse }
	mock := newMock()
	config := &Config{
		AccountRefreshFrequency: time.Hour,
		AccountRefreshCooldown:  0,
		KeyRefreshFrequency:     time.Second,
		KeyRefreshCooldown:      0,
	}
	store := testStore(mock, config)
	testbase.RunCases(t, []testbase.TestCase{
		&testbase.SuccessCase{
			`AuthorizedKeys("user1")`,
			func() (interface{}, error) { return store.AuthorizedKeys("user1") },
			testbase.ExpKeys["user1"],
		},
	})
	mock.keys["user1"] = []string{"key1"}

	mTime = mTime.Add(time.Second + time.Nanosecond)
	ch := registerCallback(config)
	// Trigger refresh.
	pulse <- mTime
	<-ch
	removeCallback()
	mock.keys["user1"] = []string{"key2"}
	testbase.RunCases(t, []testbase.TestCase{
		&testbase.SuccessCase{
			`AuthorizedKeys("user1")`,
			func() (interface{}, error) { return store.AuthorizedKeys("user1") },
			[]string{"key1"},
		},
	})
	timeAfter = time.After
	timeNow = time.Now
}

func TestUserOnDemandRefresh(t *testing.T) {
	mock := newMock()
	mock.usersGroupsError = errors.New("")
	config := &Config{
		AccountRefreshFrequency: time.Hour,
		AccountRefreshCooldown:  0,
		KeyRefreshFrequency:     time.Hour,
		KeyRefreshCooldown:      0,
	}
	store := testStore(mock, config)
	testbase.RunCases(t, []testbase.TestCase{
		&testbase.FailureCase{
			`UserByName("user1")`,
			func() (interface{}, error) { return store.UserByName("user1") },
			`unable to find user with name "user1"`,
		},
	})
	mock.usersGroupsError = nil
	ch := registerCallback(config)
	testbase.RunCases(t, []testbase.TestCase{
		&testbase.SuccessCase{
			`UserByName("user1")`,
			func() (interface{}, error) { return store.UserByName("user1") },
			testbase.ExpUsers[0],
		},
	})
	<-ch
	removeCallback()
}

func TestGroupOnDemandRefresh(t *testing.T) {
	mock := newMock()
	mock.usersGroupsError = errors.New("")
	config := &Config{
		AccountRefreshFrequency: time.Hour,
		AccountRefreshCooldown:  0,
		KeyRefreshFrequency:     time.Hour,
		KeyRefreshCooldown:      0,
	}
	store := testStore(mock, config)
	mock.usersGroupsError = nil

	ch := registerCallback(config)
	testbase.RunCases(t, []testbase.TestCase{
		&testbase.FailureCase{
			`GroupByName("group1")`,
			func() (interface{}, error) { return store.GroupByName("group1") },
			`unable to find group with name "group1"`,
		},
	})
	<-ch
	removeCallback()
	testbase.RunCases(t, []testbase.TestCase{
		&testbase.SuccessCase{
			`GroupByName("group1")`,
			func() (interface{}, error) { return store.GroupByName("group1") },
			testbase.ExpGroups[0],
		},
	})
}

func TestEmptyUsersGroups(t *testing.T) {
	emptyMock := &mockAPIClient{}
	config := &Config{time.Hour, time.Hour, time.Hour, 0}
	store := testStore(emptyMock, config)
	testbase.RunCases(t, []testbase.TestCase{
		&testbase.SuccessCase{
			"Names()",
			func() (interface{}, error) { return store.Names() },
			[]string{"gce-sudoers"},
		},
	})
}

func TestEmptyKeys(t *testing.T) {
	mock := newMock()
	emptyMock := &mockAPIClient{users: mock.users}
	config := &Config{time.Hour, time.Hour, time.Hour, 0}
	store := testStore(emptyMock, config)
	testbase.RunCases(t, []testbase.TestCase{
		&testbase.SuccessCase{
			`AuthorizedKeys("user1")`,
			func() (interface{}, error) { return store.AuthorizedKeys("user1") },
			[]string(nil),
		},
	})
}
