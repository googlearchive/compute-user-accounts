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

"""Compute Accounts service entities."""

import collections
import time


# struct passwd {
#   char   *pw_name;       /* username */
#   char   *pw_passwd;     /* user password */ <--- Unused.
#   uid_t   pw_uid;        /* user ID */
#   gid_t   pw_gid;        /* group ID */
#   char   *pw_gecos;      /* user information */
#   char   *pw_dir;        /* home directory */
#   char   *pw_shell;      /* shell program */
# };
User = collections.namedtuple('User', ['name', 'uid', 'gid', 'gecos', 'dir',
                                       'shell'])

# struct group {
#   char   *gr_name;        /* group name */
#   char   *gr_passwd;      /* group password */ <--- Unused.
#   gid_t   gr_gid;         /* group ID */
#   char  **gr_mem;         /* NULL-terminated array of pointers
#                              to names of group members */
# };
Group = collections.namedtuple('Group', ['name', 'gid', 'mem'])

AuthorizedKeys = collections.namedtuple('AuthorizedKeys', ['timestamp',
                                                           'authorized_keys'])


def json_to_user(json):
  return User(json['username'], json['uid'], json['gid'], json['gecos'],
              json['homeDirectory'], json['shell'])


def json_to_group(json):
  return Group(json['groupName'], json['gid'], tuple(json.get('members', [])))


def json_to_authorized_keys(json):
  return AuthorizedKeys(timestamp=time.time(),
                        authorized_keys=tuple(json.get('keys', [])))
