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

package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/GoogleCloudPlatform/compute-user-accounts/apiclient"
	"github.com/GoogleCloudPlatform/compute-user-accounts/logger"
	"github.com/GoogleCloudPlatform/compute-user-accounts/server"
	"github.com/GoogleCloudPlatform/compute-user-accounts/store"
)

var (
	// version is set at compile time.
	version          string
	userAgent        = fmt.Sprintf("gcua/%v", version)
	refreshFrequency = 4 * time.Minute
	refreshCooldown  = 5 * time.Second
	keyExpiration    = 30 * time.Minute
	keyCooldown      = 500 * time.Millisecond

	apiBase      = flag.String("computeaccounts", "https://www.googleapis.com/computeaccounts/alpha/", "the URL to the base of the computeaccounts API")
	instanceBase = flag.String("compute", "https://www.googleapis.com/compute/v1/", "the URL to the base of the compute API")
)

func main() {
	flag.Parse()
	logger.Info("Starting daemon.")
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt, os.Kill, syscall.SIGTERM)
	api, err := apiclient.New(&apiclient.Config{
		APIBase:      *apiBase,
		InstanceBase: *instanceBase,
		UserAgent:    userAgent,
	})
	if err != nil {
		logger.Fatalf("Init failed: %v.", err)
	}
	srv := &server.Server{store.New(api, &store.Config{
		RefreshFrequency: refreshFrequency,
		RefreshCooldown:  refreshCooldown,
		KeyExpiration:    keyExpiration,
		KeyCooldown:      keyCooldown,
	})}
	go func() {
		err := srv.Serve()
		logger.Fatalf("Server failed: %v.", err)
	}()

	for {
		select {
		case sig := <-interrupt:
			logger.Fatalf("Got interrupt: %v.", sig)
		}
	}
}
