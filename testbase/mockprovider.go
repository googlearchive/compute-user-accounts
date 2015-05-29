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

package testbase

import "github.com/GoogleCloudPlatform/compute-user-accounts/accounts"

// A MockProvider is an AccountProvider used for testing.
type MockProvider struct {
	Err  error
	Usrs []*accounts.User
	Grps []*accounts.Group
	Nams []string
	Keys map[string][]string
}

// UserByName satisfies AccountProvider.
func (m *MockProvider) UserByName(name string) (*accounts.User, error) {
	for _, u := range m.Usrs {
		if name == u.Name {
			return u, m.Err
		}
	}
	return nil, accounts.UsernameNotFound(name)
}

// UserByUID satisfies AccountProvider.
func (m *MockProvider) UserByUID(uid uint32) (*accounts.User, error) {
	for _, u := range m.Usrs {
		if uid == u.UID {
			return u, m.Err
		}
	}
	return nil, accounts.UIDNotFound(uid)
}

// Users satisfies AccountProvider.
func (m *MockProvider) Users() ([]*accounts.User, error) {
	return m.Usrs, m.Err
}

// GroupByName satisfies AccountProvider.
func (m *MockProvider) GroupByName(name string) (*accounts.Group, error) {
	for _, g := range m.Grps {
		if name == g.Name {
			return g, m.Err
		}
	}
	return nil, accounts.GroupNameNotFound(name)
}

// GroupByGID satisfies AccountProvider.
func (m *MockProvider) GroupByGID(gid uint32) (*accounts.Group, error) {
	for _, g := range m.Grps {
		if gid == g.GID {
			return g, m.Err
		}
	}
	return nil, accounts.GIDNotFound(gid)
}

// Groups satisfies AccountProvider.
func (m *MockProvider) Groups() ([]*accounts.Group, error) {
	return m.Grps, m.Err
}

// Names satisfies AccountProvider.
func (m *MockProvider) Names() ([]string, error) {
	return m.Nams, m.Err
}

// IsName satisfies AccountProvider.
func (m *MockProvider) IsName(name string) (bool, error) {
	for _, n := range m.Nams {
		if name == n {
			return true, m.Err
		}
	}
	return false, m.Err
}

// AuthorizedKeys satisfies AccountProvider.
func (m *MockProvider) AuthorizedKeys(username string) ([]string, error) {
	return m.Keys[username], m.Err
}
