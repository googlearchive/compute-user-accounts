# Copyright 2015 Google Inc. All Rights Reserved.
"""Tests for Cache."""

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
