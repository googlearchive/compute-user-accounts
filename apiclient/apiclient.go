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

// Package apiclient provides an interface for communicating with the Compute
// Accounts API.
package apiclient

import (
	"net/http"
	"path"
	"strings"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/googleapi"
	"google.golang.org/cloud/compute/metadata"

	"github.com/GoogleCloudPlatform/compute-user-accounts/logger"

	computeaccounts "google.golang.org/api/computeaccounts/v0.alpha"
)

// An APIClient allows fetching of accounts information from the Compute
// Accounts API.
type APIClient interface {
	// UsersAndGroups fetches information about all users and groups.
	UsersAndGroups() ([]*computeaccounts.LinuxUserView, []*computeaccounts.LinuxGroupView, error)
	// AuthorizedKeys fetches the authorized SSH keys for the given
	// username.
	AuthorizedKeys(username string) ([]string, error)
}

// A Config provides configuration options for an APIClient.
type Config struct {
	// APIBase is the URL of the Compute Accounts API root to communicate
	// with.
	APIBase string
	// InstanceBase is the URL of the Compute API root that the instance was
	// created with.
	InstanceBase string
	// UserAgent is the user-agent string that will be sent to the Compute
	// Accounts API.
	UserAgent string
}

type googleAPIClient struct {
	config  *Config
	service *computeaccounts.Service
}

// New creates a new APIClient with the provided configuration.
func New(config *Config) (APIClient, error) {
	service, err := computeaccounts.New(&http.Client{
		Transport: &oauth2.Transport{Source: google.ComputeTokenSource("")},
	})
	if err != nil {
		return nil, err
	}
	service.BasePath = strings.TrimRight(config.APIBase, "/") + "/projects/"
	service.UserAgent = config.UserAgent
	client := &googleAPIClient{
		config:  config,
		service: service,
	}
	return client, nil
}

// UsersAndGroups satisfies APIClient.
func (c *googleAPIClient) UsersAndGroups() ([]*computeaccounts.LinuxUserView, []*computeaccounts.LinuxGroupView, error) {
	logger.Info("Fetching users and groups.")
	p, z, i, err := c.instanceInfo()
	if err != nil {
		return nil, nil, err
	}
	view, err := c.service.Linux.GetLinuxAccountViews(p, z, i).Do()
	if err != nil {
		return nil, nil, err
	} else if view.Resource == nil {
		// No users or groups.
		return nil, nil, nil
	} else {
		return view.Resource.UserViews, view.Resource.GroupViews, nil
	}
}

// AuthorizedKeys satisfies APIClient.
func (c *googleAPIClient) AuthorizedKeys(username string) ([]string, error) {
	logger.Infof("Fetching authorized keys for %v.", username)
	p, z, i, err := c.instanceInfo()
	if err != nil {
		return nil, err
	}
	view, err := c.service.Linux.GetAuthorizedKeysView(p, z, username, i).Do()
	switch e := err.(type) {
	case nil:
		if view.Resource == nil {
			logger.Noticef("User %v has no authorized keys.", username)
			return nil, nil
		}
		return view.Resource.Keys, nil
	case *googleapi.Error:
		if e.Code == 404 {
			logger.Noticef("User %v does not exist.", username)
			return nil, nil
		}
		return nil, err
	default:
		return nil, err
	}
}

func (c *googleAPIClient) instanceInfo() (project, zone, instance string, err error) {
	project, err = metadata.ProjectID()
	if err != nil {
		return
	}
	zone, err = metadata.Zone()
	if err != nil {
		return
	}
	name, err := metadata.InstanceName()
	if err != nil {
		return
	}
	instancePath := path.Join("projects", project, "zones", zone, "instances", name)
	instance = strings.TrimRight(c.config.InstanceBase, "/") + "/" + instancePath
	return
}
