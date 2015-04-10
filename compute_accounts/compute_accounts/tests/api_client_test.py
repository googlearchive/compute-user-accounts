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

"""Tests for ApiClient."""

import re
import textwrap
import unittest

import compute_accounts
from compute_accounts.api_client import ApiClient
from compute_accounts.entities import AuthorizedKeys
from compute_accounts.entities import Group
from compute_accounts.entities import User
import mock
import responses

_GET_PROJECT_URL = ('http://metadata.google.internal/computeMetadata/v1/'
                    'project/project-id')
_GET_HOSTNAME_URL = ('http://metadata.google.internal/computeMetadata/v1/'
                     'instance/hostname')
_GET_ZONE_URL = ('http://metadata.google.internal/computeMetadata/v1/'
                 'instance/zone')
_GET_AUTH_URL = ('http://metadata.google.internal/computeMetadata/v1/instance/'
                 'service-accounts/default/token')
_LINUX_VIEWS_URL = ('https://www.googleapis.com/computeaccounts/alpha/projects/'
                    'project-1/zones/us-central1-b/linuxAccountViews')
_KEYS_VIEW_URL = ('https://www.googleapis.com/computeaccounts/alpha/projects/'
                  'project-1/zones/us-central1-b/authorizedKeysView')


class ApiClientTest(unittest.TestCase):

  # pylint: disable=invalid-name
  # assertRaisesMatch matches unittest assert naming (i.e. assertRaisesRegexp).
  def assertRaisesMatch(self, e, message):
    return self.assertRaisesRegexp(e, re.escape(message))
  # pylint: enable=invalid-name

  def _add_metadata_responses(self):
    responses.add(responses.GET, _GET_PROJECT_URL, body='project-1')
    responses.add(responses.GET, _GET_HOSTNAME_URL,
                  body='instance-1.c.project-1.internal')
    responses.add(responses.GET, _GET_ZONE_URL,
                  body='/projects/987512981027/zones/us-central1-b')
    responses.add(responses.GET, _GET_AUTH_URL,
                  body='{"access_token":"cvso3fUSxAfQ","token_type":"Bearer"}')

  def _verify_request_data(self):
    for i in range(0, 4):
      self.assertEqual(
          responses.calls[i].request.headers['Metadata-Flavor'], 'Google')
    main_request = responses.calls[4].request
    # Assert that the instance query parameter was included.
    self.assertIn('?instance=https%3A%2F%2Fwww.googleapis.com%2Fcompute%2Fv1%2F'
                  'projects%2Fproject-1%2Fzones%2Fus-central1-b%2Finstances%2F'
                  'instance-1', main_request.url)
    self.assertDictContainsSubset({'Authorization': 'Bearer cvso3fUSxAfQ'},
                                  main_request.headers)

  @responses.activate
  def test_get_users_groups_basic_case(self):
    json_response = textwrap.dedent("""\
        {
            "resource": {
                "userViews": [
                    {
                        "username": "user1",
                        "uid": 1001,
                        "gid": 1001,
                        "gecos": "",
                        "homeDirectory": "/home/user1",
                        "shell": "/bin/bash"
                    },
                    {
                        "username": "user2",
                        "uid": 1002,
                        "gid": 1001,
                        "gecos": "",
                        "homeDirectory": "/home/user2",
                        "shell": "/bin/bash"
                    }
                ],
                "groupViews": [
                    {
                        "groupName": "group1",
                        "gid": 1001,
                        "members": ["user1", "user2"]
                    },
                    {
                        "groupName": "group2",
                        "gid": 1002,
                        "members": []
                    }
                ]
            }
        }""")
    expected_users = (
        User(name='user1', uid=1001, gid=1001, gecos='', dir='/home/user1',
             shell='/bin/bash'),
        User(name='user2', uid=1002, gid=1001, gecos='', dir='/home/user2',
             shell='/bin/bash'))
    expected_groups = (
        Group(name='group1', gid=1001, mem=('user1', 'user2')),
        Group(name='group2', gid=1002, mem=()))
    self._add_metadata_responses()
    responses.add(responses.POST, _LINUX_VIEWS_URL, body=json_response)

    client = ApiClient('https://www.googleapis.com/', 'alpha', 'v1')
    result_users, result_groups = client.get_users_and_groups()
    self.assertEqual(result_users, expected_users)
    self.assertEqual(result_groups, expected_groups)
    self._verify_request_data()

  @responses.activate
  def test_get_users_groups_future_proof(self):
    json_response = textwrap.dedent("""\
        {
            "resource": {
                "userViews": [
                    {
                        "username": "user1",
                        "uid": 1001,
                        "gid": 1001,
                        "gecos": "",
                        "homeDirectory": "/home/user1",
                        "shell": "/bin/bash",
                        "futureField": 1
                    }
                ],
                "groupViews": [
                    {
                        "groupName": "group1",
                        "gid": 1001,
                        "members": ["user1", "user2"],
                        "futureField": 1
                    }
                ],
                "futureField": 1
            }
        }""")
    expected_users = (
        User(name='user1', uid=1001, gid=1001, gecos='', dir='/home/user1',
             shell='/bin/bash'),)
    expected_groups = (
        Group(name='group1', gid=1001, mem=('user1', 'user2')),)
    self._add_metadata_responses()
    responses.add(responses.POST, _LINUX_VIEWS_URL, body=json_response)

    client = ApiClient('https://www.googleapis.com/', 'alpha', 'v1')
    result_users, result_groups = client.get_users_and_groups()
    self.assertEqual(result_users, expected_users)
    self.assertEqual(result_groups, expected_groups)
    self._verify_request_data()

  @responses.activate
  def test_get_users_groups_empty(self):
    json_response = '{}'
    expected_users = ()
    expected_groups = ()
    self._add_metadata_responses()
    responses.add(responses.POST, _LINUX_VIEWS_URL, body=json_response)

    client = ApiClient('https://www.googleapis.com/', 'alpha', 'v1')
    result_users, result_groups = client.get_users_and_groups()
    self.assertEqual(result_users, expected_users)
    self.assertEqual(result_groups, expected_groups)
    self._verify_request_data()

  @responses.activate
  def test_no_users(self):
    json_response = textwrap.dedent("""\
        {
            "resource": {
                "groupViews": [
                    {
                        "groupName": "group1",
                        "gid": 1001
                    }
                ]
            }
        }""")
    expected_users = ()
    expected_groups = (
        Group(name='group1', gid=1001, mem=()),)
    self._add_metadata_responses()
    responses.add(responses.POST, _LINUX_VIEWS_URL, body=json_response)

    client = ApiClient('https://www.googleapis.com/', 'alpha', 'v1')
    result_users, result_groups = client.get_users_and_groups()
    self.assertEqual(result_users, expected_users)
    self.assertEqual(result_groups, expected_groups)
    self._verify_request_data()

  @responses.activate
  def test_metadata_failed(self):
    client = ApiClient('https://www.googleapis.com/', 'alpha', 'v1')
    # Ensure that the inner exception message is printed.
    with self.assertRaisesRegexp(
        compute_accounts.BackendException,
        'Error while sending request.\nConnectionError.*metadata'):
      client.get_users_and_groups()

  @responses.activate
  def test_get_users_groups_invalid_json(self):
    json_response = textwrap.dedent("""\
        {
            "resource": {
                "userViews": [
                    {
                        "username": "user1",
                        "uid": 1001,
        }""")
    self._add_metadata_responses()
    responses.add(responses.POST, _LINUX_VIEWS_URL, body=json_response)

    client = ApiClient('https://www.googleapis.com/', 'alpha', 'v1')
    with self.assertRaisesMatch(
        compute_accounts.BackendException,
        'Parsing JSON failed: [{}]'.format(json_response)):
      client.get_users_and_groups()

  @responses.activate
  def test_get_users_groups_missing_field(self):
    json_response = textwrap.dedent("""\
        {
            "resource": {
                "userViews": [
                    {
                        "username": "user1",
                        "uid": 1001,
                        "gid": 1001,
                        "gecos": ""
                    }
                ],
                "groupViews": []
            }
        }""")
    self._add_metadata_responses()
    responses.add(responses.POST, _LINUX_VIEWS_URL, body=json_response)

    client = ApiClient('https://www.googleapis.com/', 'alpha', 'v1')
    with self.assertRaisesMatch(
        compute_accounts.BackendException,
        'Validating JSON failed: [{u\'username\': u\'user1\', u\'gid\': 1001, '
        'u\'gecos\': u\'\', u\'uid\': 1001}]'):
      client.get_users_and_groups()

  @responses.activate
  def test_get_users_groups_invalid_value(self):
    json_response = textwrap.dedent("""\
        {
            "resource": {
                "userViews": [
                    {
                        "username": "1user",
                        "uid": 1001,
                        "gid": 1001,
                        "gecos": "",
                        "homeDirectory": "/home/user1",
                        "shell": "/bin/bash"
                    }
                ],
                "groupViews": []
            }
        }""")
    self._add_metadata_responses()
    responses.add(responses.POST, _LINUX_VIEWS_URL, body=json_response)

    client = ApiClient('https://www.googleapis.com/', 'alpha', 'v1')
    with self.assertRaisesMatch(compute_accounts.BackendException,
                                'Validating JSON failed: [1user]'):
      client.get_users_and_groups()

  @responses.activate
  @mock.patch('compute_accounts.entities.time.time', new=lambda: 123.1)
  def test_get_authorized_keys_basic_case(self):
    json_response = textwrap.dedent("""\
        {
            "resource": {
                "keys": [
                    "ssh-rsa AAAAB3NzaC1y2EAAAADAQABAQDVTRZ9YV user1@gmail.com",
                    "ssh-rsa AAAAB3NzaCDVTRZ9YV user2@gmail.com"
                ]
            }
        }""")
    expected_keys = AuthorizedKeys(timestamp=123.1, authorized_keys=(
        'ssh-rsa AAAAB3NzaC1y2EAAAADAQABAQDVTRZ9YV user1@gmail.com',
        'ssh-rsa AAAAB3NzaCDVTRZ9YV user2@gmail.com'))
    self._add_metadata_responses()
    responses.add(responses.POST, _KEYS_VIEW_URL + '/user1', body=json_response)

    client = ApiClient('https://www.googleapis.com/', 'alpha', 'v1')
    result_keys = client.get_authorized_keys('user1')
    self.assertEqual(result_keys, expected_keys)
    self._verify_request_data()

  @responses.activate
  @mock.patch('compute_accounts.entities.time.time', new=lambda: 9.9)
  def test_get_authorized_keys_empty(self):
    json_response = '{}'
    expected_keys = AuthorizedKeys(timestamp=9.9, authorized_keys=())
    self._add_metadata_responses()
    responses.add(responses.POST, _KEYS_VIEW_URL + '/user2', body=json_response)

    client = ApiClient('https://www.googleapis.com/', 'alpha', 'v1')
    result_keys = client.get_authorized_keys('user2')
    self.assertEqual(result_keys, expected_keys)
    self._verify_request_data()

  @responses.activate
  def test_get_authorized_keys_invalid_user_name(self):
    client = ApiClient('https://www.googleapis.com/', 'alpha', 'v1')
    with self.assertRaisesMatch(compute_accounts.NotFoundException,
                                'Invalid username.'):
      client.get_authorized_keys('user/../../instance')

  @responses.activate
  def test_get_authorized_keys_404(self):
    self._add_metadata_responses()
    responses.add(responses.POST, _KEYS_VIEW_URL + '/nonexistent-user',
                  status=404)

    client = ApiClient('https://www.googleapis.com/', 'alpha', 'v1')
    with self.assertRaisesMatch(
        compute_accounts.NotFoundException,
        'URL not found: [https://www.googleapis.com/computeaccounts/alpha/'
        'projects/project-1/zones/us-central1-b/authorizedKeysView/'
        'nonexistent-user?instance=https%3A%2F%2Fwww.googleapis.com%2F'
        'compute%2Fv1%2Fprojects%2Fproject-1%2Fzones%2Fus-central1-b%2F'
        'instances%2Finstance-1]'):
      client.get_authorized_keys('nonexistent-user')

  @responses.activate
  def test_get_500_server_error(self):
    self._add_metadata_responses()
    responses.add(responses.POST, _LINUX_VIEWS_URL, status=500,
                  body='Server Error')

    client = ApiClient('https://www.googleapis.com/', 'alpha', 'v1')
    with self.assertRaisesMatch(
        compute_accounts.BackendException,
        'Http error while sending request: [Server Error]'):
      client.get_users_and_groups()

  @responses.activate
  def test_get_users_groups_out_of_quota(self):
    json_response = '{}'
    self._add_metadata_responses()
    responses.add(responses.POST, _LINUX_VIEWS_URL, body=json_response)

    client = ApiClient('https://www.googleapis.com/', 'alpha', 'v1')
    for _ in range(0, 10):
      client.get_users_and_groups()
    with self.assertRaisesMatch(compute_accounts.OutOfQuotaException,
                                'No quota available'):
      client.get_users_and_groups()

  @responses.activate
  def test_get_authorized_keys_out_of_quota(self):
    json_response = '{}'
    self._add_metadata_responses()
    responses.add(responses.POST, _KEYS_VIEW_URL + '/user2', body=json_response)

    client = ApiClient('https://www.googleapis.com/', 'alpha', 'v1')
    for _ in range(0, 20):
      client.get_authorized_keys('user2')
    with self.assertRaisesMatch(compute_accounts.OutOfQuotaException,
                                'No quota available'):
      client.get_authorized_keys('user2')

  @responses.activate
  def test_url_parameters(self):
    responses.add(responses.GET, _GET_PROJECT_URL, body='project-2')
    responses.add(responses.GET, _GET_HOSTNAME_URL,
                  body='instance-2.c.project-2.internal')
    responses.add(responses.GET, _GET_ZONE_URL,
                  body='/projects/987512981027/zones/us-central2-b')
    responses.add(responses.GET, _GET_AUTH_URL,
                  body='{"access_token":"faiejoaefhoiu","token_type":"Bearer"}')
    responses.add(responses.POST, 'http://localhost/computeaccounts/beta/'
                  'projects/project-2/zones/us-central2-b/linuxAccountViews',
                  body='{"resource":{"userViews":[], "groupViews":[]}}')
    responses.add(responses.POST, 'http://localhost/computeaccounts/beta/'
                  'projects/project-2/zones/us-central2-b/authorizedKeysView/'
                  'user3',
                  body='{"resource":{"keys":[]}}')

    client = ApiClient('http://localhost/', 'beta', 'v2')
    client.get_users_and_groups('user3')
    client.get_authorized_keys('user3')

    # The two main computeaccounts requests are number 4 and number 9 because
    # for each main request four metadata requests are sent first.
    users_groups_request = responses.calls[4].request
    self.assertEqual(users_groups_request.headers['Authorization'],
                     'Bearer faiejoaefhoiu')
    self.assertIn('?instance=http%3A%2F%2Flocalhost%2Fcompute%2Fv2%2F'
                  'projects%2Fproject-2%2Fzones%2Fus-central2-b%2Finstances%2F'
                  'instance-2&user=user3', users_groups_request.url)
    authorized_keys_request = responses.calls[9].request
    self.assertIn('?instance=http%3A%2F%2Flocalhost%2Fcompute%2Fv2%2F'
                  'projects%2Fproject-2%2Fzones%2Fus-central2-b%2Finstances%2F'
                  'instance-2', authorized_keys_request.url)

  @responses.activate
  def test_name_parameter_encoded(self):
    json_response = '{}'
    self._add_metadata_responses()
    responses.add(responses.POST, _LINUX_VIEWS_URL, body=json_response)

    client = ApiClient('https://www.googleapis.com/', 'alpha', 'v1')
    client.get_users_and_groups('user&delete=1')

    main_request = responses.calls[4].request
    self.assertIn('&user=user%26delete%3D1', main_request.url)


if __name__ == '__main__':
  unittest.main()
