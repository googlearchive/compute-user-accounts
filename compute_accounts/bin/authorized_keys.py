#!/usr/bin/python
# Copyright 2015 Google Inc. All Rights Reserved.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

"""Gets authorized keys for a user and prints them to stdout."""

import sys
import compute_accounts


def main():
  assert len(sys.argv) == 2
  command = 'get_authorized_keys ' + sys.argv[1]
  # This always send a request to the API, so extend the timeout.
  for line in compute_accounts.get_account_info(command, timeout=5):
    sys.stdout.write(line)
    sys.stdout.write('\n')


if __name__ == '__main__':
  main()
