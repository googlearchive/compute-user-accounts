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

"""Tests for ProxyClient."""

import os
import socket
import threading
import unittest

import compute_accounts
from compute_accounts.proxy_client import get_account_info

_TEST_SOCKET_FILE = '/tmp/compute_accounts_proxy_client_test'


class ProxyClientTest(unittest.TestCase):

  def setUp(self):
    self._default_socket_path = (
        compute_accounts.accounts_proxy.UNIX_SOCKET_PATH)
    compute_accounts.accounts_proxy.UNIX_SOCKET_PATH = _TEST_SOCKET_FILE

    self._unlink_socket()
    self._socket = socket.socket(socket.AF_UNIX, socket.SOCK_STREAM)
    self._socket.setsockopt(socket.SOL_SOCKET, socket.SO_REUSEADDR, 1)
    self._socket.bind(_TEST_SOCKET_FILE)
    self._socket.listen(0)

  def tearDown(self):
    compute_accounts.accounts_proxy.UNIX_SOCKET_PATH = (
        self._default_socket_path)
    self._socket.close()
    self._unlink_socket()

  def _unlink_socket(self):
    try:
      os.unlink(_TEST_SOCKET_FILE)
    except OSError:
      pass

  def test_proxy_client_does_not_hang_on_connection(self):
    with self.assertRaises(socket.error):
      get_account_info('')

  def test_proxy_client_does_not_hang_on_read(self):
    hang = True
    def hanging_server():
      sock, _ = self._socket.accept()
      while hang:
        pass
      sock.close()
    thread = threading.Thread(target=hanging_server)
    thread.start()
    with self.assertRaises(socket.error):
      get_account_info('')
    hang = False
    thread.join()


if __name__ == '__main__':
  unittest.main()
