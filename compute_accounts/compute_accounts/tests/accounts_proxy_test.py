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

"""Tests for Server."""

import os
import socket
import threading
import time
import unittest

import compute_accounts
from compute_accounts.entities import AuthorizedKeys
from compute_accounts.entities import Group
from compute_accounts.entities import User
from compute_accounts.proxy_client import get_account_info
import mock
import tl.testing.thread

_TEST_SOCKET_FILE = '/tmp/compute_accounts_accounts_proxy_test'


class AccountsProxyTest(tl.testing.thread.ThreadAwareTestCase):

  def setUp(self):
    self._default_socket_path = (
        compute_accounts.accounts_proxy.UNIX_SOCKET_PATH)
    compute_accounts.accounts_proxy.UNIX_SOCKET_PATH = _TEST_SOCKET_FILE

    self._patcher = mock.patch(
        'compute_accounts.accounts_proxy.api_client.ApiClient')
    self._patcher.start()
    self._server = compute_accounts.AccountsProxy(None, None, None)
    self._api_client = self._server._api_client
    users = (
        User(name='user1', uid=1001, gid=1001, gecos='', dir='/home/user1',
             shell='/bin/bash'),
        User(name='user2', uid=1002, gid=1001, gecos='', dir='/home/user2',
             shell='/bin/bash'))
    groups = (
        Group(name='group1', gid=1001, mem=('user1', 'user2')),
        Group(name='group2', gid=1002, mem=()))
    self._api_client.get_users_and_groups.return_value = (users, groups)

  def tearDown(self):
    if self._server.is_serving():
      self._server.shutdown()
    self._patcher.stop()
    compute_accounts.accounts_proxy.UNIX_SOCKET_PATH = (
        self._default_socket_path)
    try:
      os.unlink(_TEST_SOCKET_FILE)
    except OSError:
      pass

  def report_threads_left_behind(self, threads):
    self.assertEqual(len(threads), 0, 'Some threads not cleaned up after test.')

  def _start_server(self):
    threading.Thread(target=self._server.start).start()
    self._wait_on_condition(self._server.is_serving)

  def _wait_on_condition(self, condition_lambda):
    start_time = time.time()
    while not condition_lambda():
      if time.time() - start_time > 3:
        self.fail('Failed waiting on condition: ' + str(condition_lambda))

  def test_get_user_by_name(self):
    request = 'get_user_by_name user2'
    expected_output = ['user2:1002:1001::/home/user2:/bin/bash']
    self._start_server()
    response = get_account_info(request)
    self.assertEqual(response, expected_output)
    # Make sure only one request was sent.
    self._api_client.get_users_and_groups.assert_called_once_with()

  def test_get_user_by_uid(self):
    request = 'get_user_by_uid 1001'
    expected_output = ['user1:1001:1001::/home/user1:/bin/bash']
    self._start_server()
    response = get_account_info(request)
    self.assertEqual(response, expected_output)
    # Make sure only one request was sent.
    self._api_client.get_users_and_groups.assert_called_once_with()

  def test_get_users(self):
    request = 'get_users'
    expected_output = ['user1:1001:1001::/home/user1:/bin/bash',
                       'user2:1002:1001::/home/user2:/bin/bash']
    self._start_server()
    response = get_account_info(request)
    self.assertItemsEqual(response, expected_output)
    # Make sure only one request was sent.
    self._api_client.get_users_and_groups.assert_called_once_with()

  def test_get_group_by_name(self):
    request = 'get_group_by_name group2'
    expected_output = ['group2:1002:']
    self._start_server()
    response = get_account_info(request)
    self.assertEqual(response, expected_output)
    # Make sure only one request was sent.
    self._api_client.get_users_and_groups.assert_called_once_with()

  def test_get_group_by_gid(self):
    request = 'get_group_by_gid 1001'
    expected_output = ['group1:1001:user1,user2']
    self._start_server()
    response = get_account_info(request)
    self.assertEqual(response, expected_output)
    # Make sure only one request was sent.
    self._api_client.get_users_and_groups.assert_called_once_with()

  def test_get_groups(self):
    request = 'get_groups'
    expected_output = ['group1:1001:user1,user2', 'group2:1002:']
    self._start_server()
    response = get_account_info(request)
    self.assertItemsEqual(response, expected_output)
    # Make sure only one request was sent.
    self._api_client.get_users_and_groups.assert_called_once_with()

  def test_is_account_name(self):
    request = 'is_account_name group2'
    expected_output = []
    self._start_server()
    response = get_account_info(request)
    self.assertEqual(response, expected_output)
    # Make sure only one request was sent.
    self._api_client.get_users_and_groups.assert_called_once_with()

  def test_get_shadows(self):
    request = 'get_account_names'
    expected_output = ['user1', 'user2', 'group1', 'group2']
    self._start_server()
    response = get_account_info(request)
    self.assertItemsEqual(response, expected_output)
    # Make sure only one request was sent.
    self._api_client.get_users_and_groups.assert_called_once_with()

  def test_get_user_by_name_causing_refresh(self):
    user = User(name='user3', uid=1003, gid=1001, gecos='',
                dir='/home/user3', shell='/bin/bash')
    request = 'get_user_by_name user3'
    expected_output = ['user3:1003:1001::/home/user3:/bin/bash']
    self._start_server()
    self._api_client.get_users_and_groups.return_value = ((user,), ())
    response = get_account_info(request)
    self.assertEqual(response, expected_output)
    # Make sure two requests were sent.
    self.assertEqual(self._api_client.get_users_and_groups.call_count, 2)

  def test_get_authorized_keys(self):
    keys = AuthorizedKeys(timestamp=time.time(), authorized_keys=(
        'ssh-rsa AAAAB3NzaC1y2EAAAADAQABAAABAQDVTRZ9YV user1@gmail.com',
        'ssh-rsa AAAAB3NzaCDVTRZ9YV user2@gmail.com'))
    request = 'get_authorized_keys user1'
    expected_output = [
        'ssh-rsa AAAAB3NzaC1y2EAAAADAQABAAABAQDVTRZ9YV user1@gmail.com',
        'ssh-rsa AAAAB3NzaCDVTRZ9YV user2@gmail.com']
    self._start_server()
    self._api_client.get_authorized_keys.return_value = keys
    response = get_account_info(request)
    self.assertEqual(response, expected_output)
    self._api_client.get_authorized_keys.assert_called_once_with('user1')

    # Keys should now be cached and fall back on BackendException.
    self._api_client.get_authorized_keys.side_effect = (
        compute_accounts.BackendException(None, ''))
    response = get_account_info(request)
    self.assertEqual(response, expected_output)
    self.assertEqual(self._api_client.get_authorized_keys.call_count, 2)

    # And on OutOfQuotaException.
    self._api_client.get_authorized_keys.side_effect = (
        compute_accounts.OutOfQuotaException(''))
    response = get_account_info(request)
    self.assertEqual(response, expected_output)

    # But not on NotFoundException.
    self._api_client.get_authorized_keys.side_effect = (
        compute_accounts.NotFoundException(''))
    with self.assertRaises(compute_accounts.NotFoundException):
      get_account_info(request)

  def test_background_thread_recovers_from_failure(self):
    request = 'get_users'
    expected_output1 = []
    expected_output2 = ['user2:1002:1001::/home/user2:/bin/bash',
                        'user1:1001:1001::/home/user1:/bin/bash']
    self._api_client.get_users_and_groups.side_effect = (
        compute_accounts.BackendException(None, ''))
    default_refresh = (
        compute_accounts.accounts_proxy._BACKGROUND_REFRESH_FREQUENCY_SEC)
    compute_accounts.accounts_proxy._BACKGROUND_REFRESH_FREQUENCY_SEC = 1
    self._start_server()
    response = get_account_info(request)
    self.assertEqual(response, expected_output1)
    # Remove failure.
    self._api_client.get_users_and_groups.side_effect = None
    # Make sure the refresh thread has had time to complete another cycle.
    cond = lambda: self._api_client.get_users_and_groups.call_count > 1
    self._wait_on_condition(cond)
    response = get_account_info(request)
    self.assertEqual(response, expected_output2)
    compute_accounts.accounts_proxy._BACKGROUND_REFRESH_FREQUENCY_SEC = (
        default_refresh)

  def test_fatal_error_on_background_thread_causes_main_thread_exception(self):
    default_refresh = (
        compute_accounts.accounts_proxy._BACKGROUND_REFRESH_FREQUENCY_SEC)
    compute_accounts.accounts_proxy._BACKGROUND_REFRESH_FREQUENCY_SEC = 1
    def crashing_server():
      with self.assertRaises(SystemExit):
        self._server.start()
    thread = threading.Thread(target=crashing_server)
    thread.start()
    cond = lambda: self._api_client.get_users_and_groups.call_count > 1
    self._wait_on_condition(cond)
    self._api_client.get_users_and_groups.side_effect = SystemExit()
    thread.join()
    compute_accounts.accounts_proxy._BACKGROUND_REFRESH_FREQUENCY_SEC = (
        default_refresh)

  def test_fatal_error_on_handler_thread_causes_main_thread_exception(self):
    self._api_client.get_authorized_keys.side_effect = SystemExit()
    def start_server():
      with self.assertRaises(SystemExit):
        self._server.start()
    thread = threading.Thread(target=start_server)
    thread.start()
    with self.assertRaises(compute_accounts.LookupException):
      get_account_info('get_authorized_keys user1')
    thread.join()

  def test_error_codes(self):
    self._start_server()
    with self.assertRaises(compute_accounts.NotFoundException):
      get_account_info('get_user_by_name user3')
    with self.assertRaises(compute_accounts.NotFoundException):
      get_account_info('is_account_name user3')
    with self.assertRaises(compute_accounts.LookupException):
      get_account_info('get_user_by_uid')
    with self.assertRaises(compute_accounts.LookupException):
      get_account_info('nonexistant_method')

    self._api_client.get_authorized_keys.side_effect = (
        compute_accounts.BackendException(None, ''))
    with self.assertRaises(compute_accounts.LookupException):
      get_account_info('get_authorized_keys user1')

  def test_restarting_server(self):
    self._start_server()
    self._server.shutdown()
    request = 'get_user_by_name user2'
    expected_output = ['user2:1002:1001::/home/user2:/bin/bash']
    self._start_server()
    response = get_account_info(request)
    self.assertEqual(response, expected_output)

  def test_already_serving_and_not_serving_exceptions(self):
    self._start_server()
    with self.assertRaises(RuntimeError):
      self._server.start()
    self._server.shutdown()
    with self.assertRaises(RuntimeError):
      self._server.shutdown()

  def test_client_disconnect_does_not_crash_daemon(self):
    self._start_server()
    sock = socket.socket(socket.AF_UNIX, socket.SOCK_STREAM)
    sock.connect(_TEST_SOCKET_FILE)
    sock.close()

    request = 'get_user_by_name user2'
    expected_output = ['user2:1002:1001::/home/user2:/bin/bash']
    response = get_account_info(request)
    self.assertEqual(response, expected_output)

  def test_daemon_does_not_hang_on_hung_client(self):
    default_timeout = compute_accounts.accounts_proxy._SOCKET_TIMEOUT_SEC
    compute_accounts.accounts_proxy._SOCKET_TIMEOUT_SEC = 0.01
    stop_hanging = threading.Event()
    self._start_server()
    def hung_client():
      sock = socket.socket(socket.AF_UNIX, socket.SOCK_STREAM)
      sock.connect(_TEST_SOCKET_FILE)
      stop_hanging.wait()
      sock.close()
    thread = threading.Thread(target=hung_client)
    thread.start()
    request = 'get_user_by_name user2'
    expected_output = ['user2:1002:1001::/home/user2:/bin/bash']
    response = get_account_info(request)
    self.assertEqual(response, expected_output)
    self._server.shutdown()
    stop_hanging.set()
    thread.join()
    compute_accounts.accounts_proxy._SOCKET_TIMEOUT_SEC = default_timeout


if __name__ == '__main__':
  with tl.testing.thread.ThreadJoiner(5):
    unittest.main()
