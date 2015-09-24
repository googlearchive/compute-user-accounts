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

package server

import (
	"errors"
	"io/ioutil"
	"os"
	"reflect"
	"testing"
	"time"

	"github.com/GoogleCloudPlatform/compute-user-accounts/accounts"
	"github.com/GoogleCloudPlatform/compute-user-accounts/testbase"
)

func tempFile() string {
	temp, err := ioutil.TempFile("", "")
	if err != nil {
		panic(err)
	}
	path := temp.Name()
	if err := temp.Close(); err != nil {
		panic(err)
	}
	return path
}

func startServer(mock accounts.AccountProvider) {
	server := &Server{mock}
	ch := make(chan struct{})
	listeningCallback = func() { close(ch) }
	go func() { panic(server.Serve()) }()
	<-ch
	listeningCallback = func() {}
}

func TestAll(t *testing.T) {
	socketPath = tempFile()
	mock := &testbase.MockProvider{Usrs: testbase.ExpUsers, Grps: testbase.ExpGroups, Nams: testbase.ExpNames, Keys: testbase.ExpKeys}
	client := &Client{}
	startServer(mock)
	defer os.Remove(socketPath)
	testbase.RunCases(t, []testbase.TestCase{
		&testbase.SuccessCase{
			`UserByName("user1")`,
			func() (interface{}, error) { return client.UserByName("user1") },
			testbase.ExpUsers[0],
		},
		&testbase.SuccessCase{
			"UserByUID(1002)",
			func() (interface{}, error) { return client.UserByUID(1002) },
			testbase.ExpUsers[1],
		},
		&testbase.SuccessCase{
			`GroupByName("group1")`,
			func() (interface{}, error) { return client.GroupByName("group1") },
			testbase.ExpGroups[0],
		},
		&testbase.SuccessCase{
			"GroupByGID(1001)",
			func() (interface{}, error) { return client.GroupByGID(1001) },
			testbase.ExpGroups[1],
		},
		&testbase.SuccessCase{
			"Users()",
			func() (interface{}, error) { return client.Users() },
			testbase.ExpUsers,
		},
		&testbase.SuccessCase{
			"Groups()",
			func() (interface{}, error) { return client.Groups() },
			testbase.ExpGroups,
		},
		&testbase.SuccessCase{
			"Names()",
			func() (interface{}, error) { return client.Names() },
			testbase.ExpNames,
		},
		&testbase.SuccessCase{
			`IsName("user1")`,
			func() (interface{}, error) { return client.IsName("user1") },
			true,
		},
		&testbase.SuccessCase{
			`IsName("group1")`,
			func() (interface{}, error) { return client.IsName("group1") },
			true,
		},
		&testbase.SuccessCase{
			`IsName("nil")`,
			func() (interface{}, error) { return client.IsName("nil") },
			false,
		},
		&testbase.SuccessCase{
			`AuthorizedKeys("user1")`,
			func() (interface{}, error) { return client.AuthorizedKeys("user1") },
			testbase.ExpKeys["user1"],
		},
		&testbase.SuccessCase{
			`AuthorizedKeys("user2")`,
			func() (interface{}, error) { return client.AuthorizedKeys("user2") },
			[]string{},
		},
		&testbase.FailureCase{
			`UserByName("nil")`,
			func() (interface{}, error) { return client.UserByName("nil") },
			"unable to find user or group",
		},
		&testbase.FailureCase{
			"UserByUID(2)",
			func() (interface{}, error) { return client.UserByUID(2) },
			"unable to find user or group",
		},
		&testbase.FailureCase{
			`GroupByName("nil")`,
			func() (interface{}, error) { return client.GroupByName("nil") },
			"unable to find user or group",
		},
		&testbase.FailureCase{
			"GroupByGID(1)",
			func() (interface{}, error) { return client.GroupByGID(1) },
			"unable to find user or group",
		},
	})
	mock.Err = errors.New("")
	testbase.RunCases(t, []testbase.TestCase{
		&testbase.FailureCase{
			`UserByName("user1")`,
			func() (interface{}, error) { return client.UserByName("user1") },
			"request failed",
		},
		&testbase.FailureCase{
			"UserByUID(1002)",
			func() (interface{}, error) { return client.UserByUID(1002) },
			"request failed",
		},
		&testbase.FailureCase{
			`GroupByName("group1")`,
			func() (interface{}, error) { return client.GroupByName("group1") },
			"request failed",
		},
		&testbase.FailureCase{
			"GroupByGID(1001)",
			func() (interface{}, error) { return client.GroupByGID(1001) },
			"request failed",
		},
		&testbase.FailureCase{
			"Users()",
			func() (interface{}, error) { return client.Users() },
			"request failed",
		},
		&testbase.FailureCase{
			"Groups()",
			func() (interface{}, error) { return client.Groups() },
			"request failed",
		},
		&testbase.FailureCase{
			"Names()",
			func() (interface{}, error) { return client.Names() },
			"request failed",
		},
		&testbase.FailureCase{
			`IsName("user1")`,
			func() (interface{}, error) { return client.IsName("user1") },
			"request failed",
		},
		&testbase.FailureCase{
			`IsName("group1")`,
			func() (interface{}, error) { return client.IsName("group1") },
			"request failed",
		},
		&testbase.FailureCase{
			`IsName("nil")`,
			func() (interface{}, error) { return client.IsName("nil") },
			"request failed",
		},
		&testbase.FailureCase{
			`AuthorizedKeys("user1")`,
			func() (interface{}, error) { return client.AuthorizedKeys("user1") },
			"request failed",
		},
	})
}

func TestHugeData(t *testing.T) {
	// 1000 keys ranging from 1000 bytes to ~11 Kb.
	keys := make([]string, 1000)
	for x := range keys {
		key := make([]byte, 1000+(x*10))
		var b byte
		for y := range key {
			if rune(b) == '\n' {
				continue
			}
			key[y] = b
			b++
		}
		keys[x] = string(key)
	}
	socketPath = tempFile()
	mock := &testbase.MockProvider{Usrs: testbase.ExpUsers, Keys: map[string][]string{"user1": keys}}
	client := &Client{}
	startServer(mock)
	defer os.Remove(socketPath)
	testbase.RunCases(t, []testbase.TestCase{
		&testbase.SuccessCase{
			`AuthorizedKeys("user1")`,
			func() (interface{}, error) { return client.AuthorizedKeys("user1") },
			keys,
		},
	})
}

func TestIncompleteRequests(t *testing.T) {
	socketPath = tempFile()
	mock := &testbase.MockProvider{Usrs: testbase.ExpUsers, Grps: testbase.ExpGroups, Nams: testbase.ExpNames, Keys: testbase.ExpKeys}
	startServer(mock)
	defer os.Remove(socketPath)
	testData := []struct {
		request  string
		expError string
	}{
		{"user_by_name", "request failed"},
		{"user_by_uid", "request failed"},
		{"user", "request failed"},
		{"groups_by_name ", "request failed"},
		{"groups_by_gid", "request failed"},
		{"is_name", "request failed"},
		{"keys", "request failed"},
	}
	for _, data := range testData {
		_, err := send(data.request, false)
		if err.Error() != data.expError {
			t.Errorf(`send("%v", false) = (_, %v); want (_, %v)`, data.request, err, data.expError)
		}
	}
	var defaultServerTimeout time.Duration
	defaultServerTimeout, serverTimeout = serverTimeout, time.Nanosecond
	send("", false)
	serverTimeout = defaultServerTimeout
	// Ensure server did not crash.
	resp, err := send("is_name user1", false)
	if !reflect.DeepEqual(resp, []string{}) || err != nil {
		t.Errorf(`send("is_name user1", false) = (%v, %v); want ([], nil)`, resp, err)
	}
}

func TestClientTimeout(t *testing.T) {
	var defaultClientTimeout, defaultExtendedTimeout time.Duration
	defaultClientTimeout, clientTimeout = clientTimeout, time.Nanosecond
	defaultExtendedTimeout, extendedTimeout = extendedTimeout, time.Hour
	_, err := send("", false)
	if err == nil {
		t.Errorf(`send("", false) = (_, nil); want (_, !nil)`)
	}
	clientTimeout = time.Hour
	extendedTimeout = time.Nanosecond
	_, err = send("", true)
	if err == nil {
		t.Errorf(`send("", true) = (_, nil); want (_, !nil)`)
	}
	clientTimeout = defaultClientTimeout
	extendedTimeout = defaultExtendedTimeout
}
