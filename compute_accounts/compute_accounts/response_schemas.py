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

"""Schemas defining valid responses from the Compute Accounts service."""

# Colons are forbidden in passwd and group entries.
_NSS_STRING_REGEX = '^[^:\n]*$'
_KEY_STRING_REGEX = '^[^\n]*$'
NAME_REGEX = '^[a-z][-a-z0-9_]{0,31}$'

_USER_VIEW_SCHEMA = {
    'type': 'object',
    'properties': {
        'username': {'type': 'string', 'pattern': NAME_REGEX},
        'uid': {'type': 'integer'},
        'gid': {'type': 'integer'},
        'gecos': {'type': 'string', 'pattern': _NSS_STRING_REGEX},
        'homeDirectory': {'type': 'string', 'pattern': _NSS_STRING_REGEX},
        'shell': {'type': 'string', 'pattern': _NSS_STRING_REGEX}
    },
    'required': ['username', 'uid', 'gid', 'gecos', 'homeDirectory', 'shell']
}

_GROUP_VIEW_SCHEMA = {
    'type': 'object',
    'properties': {
        'groupName': {'type': 'string', 'pattern': NAME_REGEX},
        'gid': {'type': 'integer'},
        'members': {
            'type': 'array',
            'items': {'type': 'string', 'pattern': NAME_REGEX}
        }
    },
    'required': ['groupName', 'gid']
}

LINUX_VIEWS_SCHEMA = {
    'type': 'object',
    'properties': {
        'resource': {
            'type': 'object',
            'properties': {
                'userViews': {'type': 'array', 'items': _USER_VIEW_SCHEMA},
                'groupViews': {'type': 'array', 'items': _GROUP_VIEW_SCHEMA}
            }
        }
    }
}

AUTHORIZED_KEYS_VIEW_SCHEMA = {
    'type': 'object',
    'properties': {
        'resource': {
            'type': 'object',
            'properties': {
                'keys': {
                    'type': 'array',
                    'items': {'type': 'string', 'pattern': _KEY_STRING_REGEX}
                }
            }
        }
    }
}

AUTHORIZATION_SCHEMA = {
    'type': 'object',
    'properties': {
        'token_type': {'type': 'string'},
        'access_token': {'type': 'string'},
    },
    'required': ['token_type', 'access_token']
}
