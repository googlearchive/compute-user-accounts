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
	"fmt"
	"os"

	"github.com/GoogleCloudPlatform/compute-user-accounts/logger"
	"github.com/GoogleCloudPlatform/compute-user-accounts/server"
)

func main() {
	if len(os.Args) != 2 {
		logger.Fatal("Invalid username argument to authorized keys command.")
	}
	client := &server.Client{}
	keys, err := client.AuthorizedKeys(os.Args[1])
	if err != nil {
		logger.Fatalf("Authorized keys command failed: %v.", err)
	}
	for _, k := range keys {
		fmt.Println(k)
	}
}
