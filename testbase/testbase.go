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

// Package testbase provides a unit test infrastructure and a mock
// AccountProvider.
package testbase

import (
	"reflect"
	"testing"

	"github.com/GoogleCloudPlatform/compute-user-accounts/accounts"
)

// ExpUsers is sample user data.
var ExpUsers = []*accounts.User{
	&accounts.User{
		Name:          "user1",
		UID:           4001,
		GID:           4000,
		Gecos:         "John Doe",
		HomeDirectory: "/home/user1",
		Shell:         "/bin/bash",
	},
	&accounts.User{
		Name:          "user2",
		UID:           4002,
		GID:           4000,
		Gecos:         "Jane Doe",
		HomeDirectory: "/home/user2",
		Shell:         "/bin/zsh",
	},
}

// ExpGoups is sample group data.
var ExpGroups = []*accounts.Group{
	&accounts.Group{
		Name:    "group1",
		GID:     4000,
		Members: []string(nil),
	},
	&accounts.Group{
		Name:    "group2",
		GID:     4001,
		Members: []string{"user2", "user1"},
	},
}

// ExpKeys is sample key data.
var ExpKeys = map[string][]string{
	"user1": []string{
		"ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQCbfMeFT1HFEF4NUfGY1N5xV59DcUSD0wrQ+PLshX3zNseddAq0mPRs1+rLwzH53GFzJu7p9gO2tODz1HanSSVhPXN5GuMVBA/9fZCmSkcsU18v machine1",
		"ssh-rsa AAAAB3NzaC1yc2EAAAADvUTKEXOxhMCs2MGQyRe1hWfS1wqTuhskuFlw7+iyyvgw2KQDKfJod4DiyXQpNF5361PrfduTF/T5I+tanSSVhPXN5GuMVBA/9fZCmSkcsU18v machine2",
	},
	"user3": []string{
		"ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQCbfMeFT1HFEF4NUfGY1N5xV59DcUSD0wrQ+p9gO2tODz1HanSSVhPXN5GuMVBA/9fZCmSkcsU18v machine3",
		"ssh-rsa AAAAB3NzaC1yc2EAAAADvUTKEXOxhMCs2MGQyRe1hWfS1wqTuhskuFlw7+iyyvgwuTF/T5I+tanSSVhPXN5GuMVBA/9fZCmSkcsU18v machine4",
	},
}

// ExpNames is sample name data.
var ExpNames = []string{"group1", "group2", "user1", "user2"}

// A TestCase is any test case.
type TestCase interface {
	run(t *testing.T)
}

// A SuccessCase is a test case which is expected to return successfully.
type SuccessCase struct {
	Name string
	Call func() (interface{}, error)
	Exp  interface{}
}

// A FailureCase is a test case which is expected to return an error.
type FailureCase struct {
	Name string
	Call func() (interface{}, error)
	Exp  string
}

func (c *SuccessCase) run(t *testing.T) {
	ret, err := c.Call()
	if !reflect.DeepEqual(c.Exp, ret) || err != nil {
		t.Errorf("%v = (%#v, %#v); want (%#v, <nil>)", c.Name, ret, err, c.Exp)
	}
}

func (c *FailureCase) run(t *testing.T) {
	_, err := c.Call()
	if err == nil || c.Exp != err.Error() {
		t.Errorf("%v = (_, %v); want (_, %v)", c.Name, err, c.Exp)
	}
}

// RunCases runs a set of test cases.
func RunCases(t *testing.T, cases []TestCase) {
	for _, c := range cases {
		c.run(t)
	}
}
