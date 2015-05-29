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

// Package server provides a client and server for communicating account data
// over a socket.
package server

import (
	"bytes"
	"errors"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/GoogleCloudPlatform/compute-user-accounts/accounts"
	"github.com/GoogleCloudPlatform/compute-user-accounts/logger"
)

const maxRequestSize = 128

var (
	// socketPath is set through the Makefile at compile time.
	socketPath string
	// listeningCallback exposes a testing callback invoked when the server is
	// listening.
	listeningCallback = func() {}
	serverTimeout     = time.Second
)

// A Server provides account information to a Client through a socket.
type Server struct {
	Provider accounts.AccountProvider
}

// Serve begins serving accounts information through a socket forever.
func (s *Server) Serve() error {
	os.Remove(socketPath)
	sock, err := net.ListenUnix("unix", &net.UnixAddr{socketPath, "unix"})
	if err != nil {
		return err
	}
	// Make the socket readable and writeable by all.
	os.Chmod(socketPath, os.ModePerm)
	listeningCallback()
	logger.Infof("Listening for connections at %v.", socketPath)
	for {
		conn, err := sock.Accept()
		if err != nil {
			logger.Errorf("Failed to accept connection: %v.", err)
			continue
		}
		logger.Info("Accepted connection.")
		go s.handle(conn)
	}
}

func (s *Server) handle(conn net.Conn) {
	defer conn.Close()
	deadline := time.Now().Add(serverTimeout)
	conn.SetReadDeadline(deadline)
	data := make([]byte, maxRequestSize)
	n, err := conn.Read(data)
	if err != nil {
		logger.Errorf("Failed to read request: %v.", err)
		return
	}
	resp := s.respond(string(data[:n]))
	deadline = time.Now().Add(serverTimeout)
	conn.SetWriteDeadline(deadline)
	_, err = conn.Write([]byte(resp))
	if err != nil {
		logger.Errorf("Failed to write response: %v.", err)
	}
}

func (s *Server) respond(req string) string {
	parts := strings.Split(req, " ")
	cmd := parts[0]
	args := parts[1:]
	switch cmd {
	case "user_by_name":
		return s.userByName(args)
	case "user_by_uid":
		return s.userByUID(args)
	case "users":
		return s.users()
	case "group_by_name":
		return s.groupByName(args)
	case "group_by_gid":
		return s.groupByGID(args)
	case "groups":
		return s.groups()
	case "names":
		return s.names()
	case "is_name":
		return s.isName(args)
	case "keys":
		return s.authorizedKeys(args)
	default:
		logger.Errorf("Invalid request: %v.", req)
		return "400"
	}
}

func (s *Server) userByName(args []string) string {
	name, err := parseName(args)
	if err != nil {
		logger.Errorf("Invalid name for user: %v.", err)
		return "400"
	}
	logger.Infof("Getting user by name: %v.", name)
	user, err := s.Provider.UserByName(name)
	if err != nil {
		return marshalError(err)
	}
	logger.Info("Request succeeded.")
	return fmt.Sprintf("200\n%v", marshalUser(user))
}

func (s *Server) userByUID(args []string) string {
	uid, err := parseID(args)
	if err != nil {
		logger.Errorf("Invalid UID for user: %v.", err)
		return "400"
	}
	logger.Infof("Getting user by UID: %v.", uid)
	user, err := s.Provider.UserByUID(uid)
	if err != nil {
		return marshalError(err)
	}
	logger.Info("Request succeeded.")
	return fmt.Sprintf("200\n%v", marshalUser(user))
}

func (s *Server) users() string {
	logger.Info("Getting users.")
	users, err := s.Provider.Users()
	if err != nil {
		return marshalError(err)
	}
	var buf bytes.Buffer
	buf.WriteString("200")
	for _, u := range users {
		buf.WriteString("\n")
		buf.WriteString(marshalUser(u))
	}
	logger.Info("Request succeeded.")
	return buf.String()
}

func (s *Server) groupByName(args []string) string {
	name, err := parseName(args)
	if err != nil {
		logger.Errorf("Invalid name for group: %v.", err)
		return "400"
	}
	logger.Infof("Getting group by name: %v.", name)
	group, err := s.Provider.GroupByName(name)
	if err != nil {
		return marshalError(err)
	}
	logger.Info("Request succeeded.")
	return fmt.Sprintf("200\n%v", marshalGroup(group))
}

func (s *Server) groupByGID(args []string) string {
	gid, err := parseID(args)
	if err != nil {
		logger.Errorf("Invalid GID for group: %v.", err)
		return "400"
	}
	logger.Infof("Getting group by GID: %v.", gid)
	group, err := s.Provider.GroupByGID(gid)
	if err != nil {
		return marshalError(err)
	}
	logger.Info("Request succeeded.")
	return fmt.Sprintf("200\n%v", marshalGroup(group))
}

func (s *Server) groups() string {
	logger.Info("Getting groups.")
	groups, err := s.Provider.Groups()
	if err != nil {
		return marshalError(err)
	}
	var buf bytes.Buffer
	buf.WriteString("200")
	for _, g := range groups {
		buf.WriteString("\n")
		buf.WriteString(marshalGroup(g))
	}
	logger.Info("Request succeeded.")
	return buf.String()
}

func (s *Server) names() string {
	logger.Info("Getting names.")
	names, err := s.Provider.Names()
	if err != nil {
		return marshalError(err)
	}
	var buf bytes.Buffer
	buf.WriteString("200")
	for _, n := range names {
		buf.WriteString("\n")
		buf.WriteString(n)
	}
	logger.Info("Request succeeded.")
	return buf.String()
}

func (s *Server) isName(args []string) string {
	name, err := parseName(args)
	if err != nil {
		logger.Errorf("Invalid name: %v.", err)
		return "400"
	}
	logger.Infof("Checking name: %v.", name)
	is, err := s.Provider.IsName(name)
	if err != nil {
		return marshalError(err)
	} else if is {
		logger.Info("Valid name.")
		return "200"
	} else {
		logger.Info("Invalid name.")
		return "404"
	}
}

func (s *Server) authorizedKeys(args []string) string {
	username, err := parseName(args)
	if err != nil {
		logger.Errorf("Invalid username for keys: %v.", err)
		return "400"
	}
	logger.Infof("Getting keys for user: %v.", username)
	keys, err := s.Provider.AuthorizedKeys(username)
	if err != nil {
		return marshalError(err)
	}
	var buf bytes.Buffer
	buf.WriteString("200")
	for _, k := range keys {
		buf.WriteString("\n")
		buf.WriteString(k)
	}
	logger.Info("Request succeeded.")
	return buf.String()
}

func parseName(args []string) (string, error) {
	if len(args) == 0 {
		return "", errors.New("no args")
	}
	return args[0], nil
}

func parseID(args []string) (uint32, error) {
	if len(args) == 0 {
		return 0, errors.New("no args")
	}
	val, err := strconv.ParseUint(args[0], 10, 0)
	return uint32(val), err
}

func marshalError(err error) string {
	switch err.(type) {
	case *accounts.NotFoundError:
		logger.Noticef("Request failed: %v.", err)
		return "404"
	default:
		logger.Noticef("Request failed: %v.", err)
		return "500"
	}
}

func marshalUser(user *accounts.User) string {
	uid := strconv.FormatUint(uint64(user.UID), 10)
	gid := strconv.FormatUint(uint64(user.GID), 10)
	return strings.Join([]string{user.Name, uid, gid, user.Gecos, user.HomeDirectory, user.Shell}, ":")
}

func marshalGroup(group *accounts.Group) string {
	mem := strings.Join(group.Members, ",")
	gid := strconv.FormatUint(uint64(group.GID), 10)
	return strings.Join([]string{group.Name, gid, mem}, ":")
}
