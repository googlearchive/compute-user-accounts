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

// Package accounts provides an interface, structs, and errors needed for
// providing User and Group account information.
package accounts

import "fmt"

// A User is a Linux user account.
type User struct {
	Name          string
	UID           uint32
	GID           uint32
	Gecos         string
	HomeDirectory string
	Shell         string
}

// A Group is a Linux group.
type Group struct {
	Name    string
	GID     uint32
	Members []string
}

// An AccountProvider provides information about users and groups.
type AccountProvider interface {
	// UserByName fetches information about a user by searching for a
	// username.
	UserByName(name string) (*User, error)
	// UserByUID fetches information about a user by searching for a
	// UID.
	UserByUID(uid uint32) (*User, error)
	// Users fetches information about all users known by the
	// AccountProvider.
	Users() ([]*User, error)
	// GroupByName fetches information about a group by searching for a
	// group name.
	GroupByName(name string) (*Group, error)
	// GroupByGID fetches information about a group by searching for a
	// GID.
	GroupByGID(gid uint32) (*Group, error)
	// Groups fetches information about all groups known by the
	// AccountProvider.
	Groups() ([]*Group, error)
	// Names fetches the names of all users and groups known by the
	// AccountProvider.
	Names() ([]string, error)
	// IsName returns whether or not the given name if a valid name of a
	// user/group.
	IsName(name string) (bool, error)
	// AuthorizedKeys returns the authorized SSH keys for the given
	// username.
	AuthorizedKeys(username string) ([]string, error)
}

// A NotFoundError reports that the user or group that was searched for does
// not exist.
type NotFoundError struct {
	hasDetails     bool
	entityName     string
	identifierName string
	identifier     interface{}
}

func (e *NotFoundError) Error() string {
	if !e.hasDetails {
		return "unable to find user or group"
	}
	return fmt.Sprintf("unable to find %v with %v %v", e.entityName, e.identifierName, e.identifier)
}

// UsernameNotFound returns a NotFoundError for a missing user searched by name.
func UsernameNotFound(name string) error {
	return &NotFoundError{
		hasDetails:     true,
		entityName:     "user",
		identifierName: "name",
		identifier:     fmt.Sprintf("\"%v\"", name),
	}
}

// UIDNotFound returns a NotFoundError for a missing user searched by UID.
func UIDNotFound(uid uint32) error {
	return &NotFoundError{
		hasDetails:     true,
		entityName:     "user",
		identifierName: "UID",
		identifier:     uid,
	}
}

// GroupNameNotFound returns a NotFoundError for a missing group searched by
// name.
func GroupNameNotFound(name string) error {
	return &NotFoundError{
		hasDetails:     true,
		entityName:     "group",
		identifierName: "name",
		identifier:     fmt.Sprintf("\"%v\"", name),
	}
}

// GIDNotFound returns a NotFoundError for a missing group searched by GID.
func GIDNotFound(gid uint32) error {
	return &NotFoundError{
		hasDetails:     true,
		entityName:     "group",
		identifierName: "GID",
		identifier:     gid,
	}
}
