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
	"fmt"
	"io/ioutil"
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/GoogleCloudPlatform/compute-user-accounts/accounts"
	"github.com/GoogleCloudPlatform/compute-user-accounts/logger"
)

var clientTimeout = time.Second
var extendedTimeout = 5 * time.Second

// Client implements AccountProvider as a client that sends requests to
// the Google Compute User Accounts daemon.
type Client struct{}

// UserByName satisfies AccountProvider.
func (c *Client) UserByName(name string) (*accounts.User, error) {
	return c.user(fmt.Sprintf("user_by_name %v", name), true)
}

// UserByUID satisfies AccountProvider.
func (c *Client) UserByUID(uid uint32) (*accounts.User, error) {
	return c.user(fmt.Sprintf("user_by_uid %v", uid), false)
}

func (c *Client) user(cmd string, extendTimeout bool) (*accounts.User, error) {
	resp, err := send(cmd, extendTimeout)
	if err != nil {
		return nil, err
	} else if len(resp) == 0 {
		return nil, errors.New("no user in response")
	} else {
		return unmarshalUser(resp[0])
	}
}

// Users satisfies AccountProvider.
func (c *Client) Users() ([]*accounts.User, error) {
	resp, err := send("users", false)
	if err != nil {
		return nil, err
	}
	users := make([]*accounts.User, len(resp))
	for i, l := range resp {
		users[i], err = unmarshalUser(l)
		if err != nil {
			return nil, err
		}
	}
	return users, nil
}

// GroupByName satisfies AccountProvider.
func (c *Client) GroupByName(name string) (*accounts.Group, error) {
	return c.group(fmt.Sprintf("group_by_name %v", name))
}

// GroupByGID satisfies AccountProvider.
func (c *Client) GroupByGID(gid uint32) (*accounts.Group, error) {
	return c.group(fmt.Sprintf("group_by_gid %v", gid))
}

func (c *Client) group(cmd string) (*accounts.Group, error) {
	resp, err := send(cmd, false)
	if err != nil {
		return nil, err
	} else if len(resp) == 0 {
		return nil, errors.New("no group in response")
	} else {
		return unmarshalGroup(resp[0])
	}
}

// Groups satisfies AccountProvider.
func (c *Client) Groups() ([]*accounts.Group, error) {
	resp, err := send("groups", false)
	if err != nil {
		return nil, err
	}
	groups := make([]*accounts.Group, len(resp))
	for i, l := range resp {
		groups[i], err = unmarshalGroup(l)
		if err != nil {
			return nil, err
		}
	}
	return groups, nil
}

// Names satisfies AccountProvider.
func (c *Client) Names() ([]string, error) {
	return send("names", false)
}

// IsName satisfies AccountProvider.
func (c *Client) IsName(name string) (bool, error) {
	_, err := send(fmt.Sprintf("is_name %v", name), false)
	switch err.(type) {
	case nil:
		return true, nil
	case *accounts.NotFoundError:
		return false, nil
	default:
		return false, err
	}
}

// AuthorizedKeys satisfies AccountProvider.
func (c *Client) AuthorizedKeys(username string) ([]string, error) {
	return send(fmt.Sprintf("keys %v", username), true)
}

func send(req string, extendTimeout bool) ([]string, error) {
	conn, err := net.DialUnix("unix", nil, &net.UnixAddr{socketPath, "unix"})
	if err != nil {
		return nil, err
	}
	defer conn.Close()
	writeDeadline := time.Now().Add(clientTimeout)
	conn.SetWriteDeadline(writeDeadline)
	_, err = conn.Write([]byte(req))
	if err != nil {
		logger.Errorf("Failed to write request: %v.", err)
		return nil, err
	}
	readDeadline := time.Now().Add(clientTimeout)
	if extendTimeout {
		readDeadline = time.Now().Add(extendedTimeout)
	}
	conn.SetReadDeadline(readDeadline)
	data, err := ioutil.ReadAll(conn)
	if err != nil {
		logger.Errorf("Failed to read response: %v.", err)
		return nil, err
	}
	resp := strings.Split(string(data), "\n")
	err = checkHeader(resp)
	if err != nil {
		logger.Errorf("Request failed: %v.", err)
		return nil, err
	}
	return resp[1:], nil
}

func checkHeader(resp []string) error {
	if len(resp) == 0 {
		return errors.New("incomplete response")
	}
	switch resp[0] {
	case "200":
		return nil
	case "404":
		return &accounts.NotFoundError{}
	default:
		return errors.New("request failed")
	}
}

func unmarshalUser(line string) (*accounts.User, error) {
	parts := strings.Split(line, ":")
	if len(parts) != 6 {
		return nil, fmt.Errorf("invalid user in response: %v", line)
	}
	uid, err := strconv.ParseUint(parts[1], 10, 0)
	if err != nil {
		return nil, fmt.Errorf("invalid UID in response: %v", line)
	}
	gid, err := strconv.ParseUint(parts[2], 10, 0)
	if err != nil {
		return nil, fmt.Errorf("invalid GID in response: %v", line)
	}
	return &accounts.User{
		Name:          parts[0],
		UID:           uint32(uid),
		GID:           uint32(gid),
		Gecos:         parts[3],
		HomeDirectory: parts[4],
		Shell:         parts[5],
	}, nil
}

func unmarshalGroup(line string) (*accounts.Group, error) {
	parts := strings.Split(line, ":")
	if len(parts) != 3 {
		return nil, fmt.Errorf("invalid group in response: %v", line)
	}
	gid, err := strconv.ParseUint(parts[1], 10, 0)
	if err != nil {
		return nil, fmt.Errorf("invalid GID in response: %v", line)
	}
	var members []string
	if parts[2] != "" {
		members = strings.Split(parts[2], ",")
	}
	return &accounts.Group{
		Name:    parts[0],
		GID:     uint32(gid),
		Members: members,
	}, nil
}
