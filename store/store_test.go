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
	"reflect"
	"sort"
	"testing"
	"time"

	"github.com/GoogleCloudPlatform/compute-user-accounts/accounts"
	"github.com/GoogleCloudPlatform/compute-user-accounts/testbase"

	_ "net/http/pprof"

	cua "google.golang.org/api/clouduseraccounts/vm_alpha"
)

type mockAPIClient struct {
	users            []*cua.LinuxUserView
	groups           []*cua.LinuxGroupView
	usersGroupsError error
	usersGroupsCount uint8
	keys             map[string][]string
	keysError        error
	keysLastUser     string
	keysCount        uint8
}

// UsersAndGroups satisfies APIClient.
func (c *mockAPIClient) UsersAndGroups() ([]*cua.LinuxUserView, []*cua.LinuxGroupView, error) {
	c.usersGroupsCount++
	return c.users, c.groups, c.usersGroupsError
}

// UsersAndGroups satisfies APIClient.
func (c *mockAPIClient) AuthorizedKeys(username string) ([]string, error) {
	c.keysCount++
	keys, ok := c.keys[username]
	if ok {
		return keys, c.keysError
	}
	// This case is equivalent to an API 404, return the nil slice.
	return nil, c.keysError
}

func (c *mockAPIClient) AssertCalls(t *testing.T, expUG, expKey uint8) {
	if c.usersGroupsCount != expUG {
		t.Errorf("UsersAndGroups() count %v; want %v", c.usersGroupsCount, expUG)
	}
	if c.keysCount != expKey {
		t.Errorf("AuthorizedKeys(_) count %v; want %v", c.keysCount, expKey)
	}
}

func (c *mockAPIClient) Clear() {
	c.usersGroupsCount = 0
	c.keysCount = 0
	c.usersGroupsError = nil
	c.keysError = nil
}

type userSlice []*accounts.User

func (s userSlice) Len() int           { return len(s) }
func (s userSlice) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }
func (s userSlice) Less(i, j int) bool { return s[i].Name < s[j].Name }

type groupSlice []*accounts.Group

func (s groupSlice) Len() int           { return len(s) }
func (s groupSlice) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }
func (s groupSlice) Less(i, j int) bool { return s[i].Name < s[j].Name }

var mock = &mockAPIClient{
	users: []*cua.LinuxUserView{
		&cua.LinuxUserView{
			Username:      "user1",
			Uid:           4001,
			Gid:           4000,
			Gecos:         "John Doe",
			HomeDirectory: "/home/user1",
			Shell:         "/bin/bash",
		},
		&cua.LinuxUserView{
			Username:      "user2",
			Uid:           4002,
			Gid:           4000,
			Gecos:         "Jane Doe",
			HomeDirectory: "/home/user2",
			Shell:         "/bin/zsh",
		},
	},
	groups: []*cua.LinuxGroupView{
		&cua.LinuxGroupView{
			GroupName: "group1",
			Gid:       4000,
			Members:   []string(nil),
		},
		&cua.LinuxGroupView{
			GroupName: "group2",
			Gid:       4001,
			Members:   []string{"user2", "user1"},
		},
	},
	keys: testbase.ExpKeys,
}

func TestUsersGroups(t *testing.T) {
	mock.Clear()
	config := &Config{time.Hour, time.Hour, time.Hour, 0}
	store := New(mock, config)
	testbase.RunCases(t, []testbase.TestCase{
		&testbase.SuccessCase{
			`UserByName("user1")`,
			func() (interface{}, error) { return store.UserByName("user1") },
			testbase.ExpUsers[0],
		},
		&testbase.SuccessCase{
			"UserByUID(4002)",
			func() (interface{}, error) { return store.UserByUID(4002) },
			testbase.ExpUsers[1],
		},
		&testbase.SuccessCase{
			`GroupByName("group1")`,
			func() (interface{}, error) { return store.GroupByName("group1") },
			testbase.ExpGroups[0],
		},
		&testbase.SuccessCase{
			"GroupByGID(4001)",
			func() (interface{}, error) { return store.GroupByGID(4001) },
			testbase.ExpGroups[1],
		},
		&testbase.SuccessCase{
			"Users()",
			func() (interface{}, error) { r, e := store.Users(); sort.Sort(userSlice(r)); return r, e },
			testbase.ExpUsers,
		},
		&testbase.SuccessCase{
			"Groups()",
			func() (interface{}, error) { r, e := store.Groups(); sort.Sort(groupSlice(r)); return r, e },
			testbase.ExpGroups,
		},
		&testbase.SuccessCase{
			"Names()",
			func() (interface{}, error) { r, e := store.Names(); sort.Sort(sort.StringSlice(r)); return r, e },
			testbase.ExpNames,
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
	mock.AssertCalls(t, 1, 0)
}

func TestKeysBasicCase(t *testing.T) {
	mock.Clear()
	config := &Config{time.Hour, time.Hour, time.Hour, 0}
	store := New(mock, config)
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
	mock.AssertCalls(t, 1, 2)
}

func TestUserOnDemandRefresh(t *testing.T) {
	mock.Clear()
	mock.usersGroupsError = errors.New("")
	// No cooldown.
	config := &Config{time.Hour, 0, time.Hour, 0}
	store := New(mock, config)
	testbase.RunCases(t, []testbase.TestCase{
		&testbase.FailureCase{
			`UserByName("user1")`,
			func() (interface{}, error) { return store.UserByName("user1") },
			`unable to find user with name "user1"`,
		},
	})
	mock.usersGroupsError = nil
	testbase.RunCases(t, []testbase.TestCase{
		&testbase.SuccessCase{
			`UserByName("user1")`,
			func() (interface{}, error) { return store.UserByName("user1") },
			testbase.ExpUsers[0],
		},
	})
	// First update and twice for missing username.
	mock.AssertCalls(t, 3, 0)
}

func TestGroupOnDemandRefresh(t *testing.T) {
	mock.Clear()
	mock.usersGroupsError = errors.New("")
	// No cooldown.
	config := &Config{time.Hour, 0, time.Hour, 0}
	store := New(mock, config)
	mock.usersGroupsError = nil
	// Cache refresh callback.
	ch := make(chan struct{})
	refreshCallback = func() { close(ch) }
	testbase.RunCases(t, []testbase.TestCase{
		&testbase.FailureCase{
			`GroupByName("group1")`,
			func() (interface{}, error) { return store.GroupByName("group1") },
			`unable to find group with name "group1"`,
		},
	})
	<-ch
	refreshCallback = func() {}
	testbase.RunCases(t, []testbase.TestCase{
		&testbase.SuccessCase{
			`GroupByName("group1")`,
			func() (interface{}, error) { return store.GroupByName("group1") },
			testbase.ExpGroups[0],
		},
	})
	// First update and once for missing username.
	mock.AssertCalls(t, 2, 0)
}

func TestCachedKeyCooldownAndExpiration(t *testing.T) {
	mock.Clear()
	// Keys expire after two seconds and dont fetch again for one second.
	config := &Config{time.Hour, time.Hour, 2 * time.Second, time.Second}
	store := New(mock, config)
	mTime := time.Now().UTC()
	// Mock time.
	utcTime = func() time.Time { return mTime }
	testbase.RunCases(t, []testbase.TestCase{
		&testbase.SuccessCase{
			`AuthorizedKeys("user1")`,
			func() (interface{}, error) { return store.AuthorizedKeys("user1") },
			testbase.ExpKeys["user1"],
		},
	})
	// This should not result in a call to the API.
	testbase.RunCases(t, []testbase.TestCase{
		&testbase.SuccessCase{
			`AuthorizedKeys("user1")`,
			func() (interface{}, error) { return store.AuthorizedKeys("user1") },
			testbase.ExpKeys["user1"],
		},
	})
	mTime = mTime.Add(time.Second + time.Nanosecond)
	mock.keysError = errors.New("API error")
	testbase.RunCases(t, []testbase.TestCase{
		&testbase.SuccessCase{
			`AuthorizedKeys("user1")`,
			func() (interface{}, error) { return store.AuthorizedKeys("user1") },
			testbase.ExpKeys["user1"],
		},
	})
	mTime = mTime.Add(time.Second)
	testbase.RunCases(t, []testbase.TestCase{
		&testbase.FailureCase{
			`AuthorizedKeys("user1")`,
			func() (interface{}, error) { return store.AuthorizedKeys("user1") },
			"API error",
		},
	})
	mock.AssertCalls(t, 1, 3)
}

func TestRefreshKeyRemoval(t *testing.T) {
	mock.Clear()
	// Mock time.
	mTime := time.Now().UTC()
	utcTime = func() time.Time { return mTime }
	pulse := make(chan time.Time)
	pulseAfter = func(d time.Duration) <-chan time.Time { return pulse }
	// Refresh after a second, no cooldown, expire keys after a second.
	config := &Config{time.Second, 0, time.Second, 0}
	store := New(mock, config)
	testbase.RunCases(t, []testbase.TestCase{
		&testbase.SuccessCase{
			`AuthorizedKeys("user1")`,
			func() (interface{}, error) { return store.AuthorizedKeys("user1") },
			testbase.ExpKeys["user1"],
		},
	})
	keyMap := reflect.ValueOf(store).Elem().FieldByName("keysByUsername")
	if keyMap.Len() != 1 {
		t.Errorf("key not cached")
	}
	// Cache refresh callback.
	ch := make(chan struct{})
	refreshCallback = func() { close(ch) }
	mTime = mTime.Add(time.Second + time.Nanosecond)
	pulse <- mTime
	<-ch
	refreshCallback = func() {}
	if keyMap.Len() != 0 {
		t.Errorf("key not removed")
	}
	mock.AssertCalls(t, 2, 1)
}

func TestEmptyUsersGroups(t *testing.T) {
	emptyMock := &mockAPIClient{}
	config := &Config{time.Hour, time.Hour, time.Hour, 0}
	store := New(emptyMock, config)
	testbase.RunCases(t, []testbase.TestCase{
		&testbase.SuccessCase{
			"Names()",
			func() (interface{}, error) { return store.Names() },
			[]string{},
		},
	})
	emptyMock.AssertCalls(t, 1, 0)
}

func TestEmptyKeys(t *testing.T) {
	emptyMock := &mockAPIClient{users: mock.users}
	config := &Config{time.Hour, time.Hour, time.Hour, 0}
	store := New(emptyMock, config)
	testbase.RunCases(t, []testbase.TestCase{
		&testbase.SuccessCase{
			`AuthorizedKeys("user1")`,
			func() (interface{}, error) { return store.AuthorizedKeys("user1") },
			[]string(nil),
		},
	})
	emptyMock.AssertCalls(t, 1, 1)
}
