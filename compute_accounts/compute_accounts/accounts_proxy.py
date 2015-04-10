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

"""Server providing Compute Accounts information via a Unix socket."""

import logging
import os
import Queue
import socket
import SocketServer
import threading
import time

from . import api_client
from . import cache
from . import exceptions

UNIX_SOCKET_PATH = '/var/run/compute_accounts/sock'
_BACKGROUND_REFRESH_FREQUENCY_SEC = 1800  # Once every half hour.
_SOCKET_TIMEOUT_SEC = 1


class AccountsProxy(SocketServer.ThreadingUnixStreamServer, object):
  """Server providing Compute Accounts information via a Unix socket."""

  def __init__(self, api_root, computeaccounts_api_version,
               compute_api_version):
    """Inits the AccountsProxy to send request to the Compute Accounts API."""
    self._unlink_socket()
    super(AccountsProxy, self).__init__(UNIX_SOCKET_PATH, RequestHandler)

    self._logger = logging.getLogger(__name__)
    self._is_serving = threading.Event()
    self._is_shutting_down = threading.Event()
    self._worker_exception_queue = Queue.Queue()

    self._api_client = api_client.ApiClient(
        api_root, computeaccounts_api_version, compute_api_version)
    self._cache = cache.Cache()

  def start(self):
    """Starts the AccountsProxy server."""
    self._logger.info('Starting Compute Accounts proxy server.')

    if self._is_serving.is_set():
      raise RuntimeError('Already serving.')

    self._refresh_cache()
    try:
      self._is_shutting_down.clear()
      self._refresh_thread = threading.Thread(target=self._refresh_thread_main)
      self._refresh_thread.start()
      self._is_serving.set()
      self.serve_forever()
    finally:
      self._is_serving.clear()
      self._is_shutting_down.set()
      self._refresh_thread.join()

    if not self._worker_exception_queue.empty():
      raise self._worker_exception_queue.get()

  def shutdown(self):
    """Cleanly shuts down the AccountsProxy server."""
    self._logger.info('Shutting down Compute Accounts proxy server.')

    if not self._is_serving.is_set():
      raise RuntimeError('Not serving.')

    super(AccountsProxy, self).shutdown()
    self._is_shutting_down.wait()
    self._refresh_thread.join()

  def is_serving(self):
    return self._is_serving.is_set()

  def _unlink_socket(self):
    try:
      os.unlink(UNIX_SOCKET_PATH)
    except OSError:
      pass

  def _refresh_cache(self):
    try:
      self._logger.info('Refreshing users and groups.')
      users, groups = self._api_client.get_users_and_groups()
      self._cache.repopulate_users_and_groups(users, groups)
    except exceptions.LookupException:
      self._logger.exception('Error during refresh.')

  def _refresh_thread_main(self):
    self._is_shutting_down.wait(_BACKGROUND_REFRESH_FREQUENCY_SEC)
    while not self._is_shutting_down.is_set():
      try:
        self._refresh_cache()
      except BaseException as e:
        self._logger.exception('Unrecoverable error during refresh.')
        self._handle_worker_exception(e)
        return
      self._is_shutting_down.wait(_BACKGROUND_REFRESH_FREQUENCY_SEC)

  def _handle_worker_exception(self, exception):
    self._worker_exception_queue.put(exception)
    super(AccountsProxy, self).shutdown()


class RequestHandler(SocketServer.BaseRequestHandler, object):
  """Class for handling requests sent to the AccountsProxy server."""

  def setup(self):
    """Sets up the request handler."""
    super(RequestHandler, self).setup()
    self._socket = self.request
    # pylint: disable=protected-access
    self._logger = logging.getLogger(__name__)
    self._cache = self.server._cache
    self._api_client = self.server._api_client
    self._exception_handler = self.server._handle_worker_exception
    # pylint: enable=protected-access

  def _user_to_passwd_line(self, user):
    """Returns a user represented as an /etc/passwd line (without password)."""
    return ':'.join([user.name, str(user.uid), str(user.gid), user.gecos,
                     user.dir, user.shell])

  def _group_to_group_line(self, group):
    """Returns a group represented as an /etc/group line (without password)."""
    members = ','.join(group.mem)
    return ':'.join([group.name, str(group.gid), members])

  def _get_user_by_name(self, name):
    """Returns user information by name. Refreshes cached data if missing."""
    self._logger.info('Getting user by name: [%s]', name)
    try:
      return [self._user_to_passwd_line(self._cache.get_user_by_name(name))]
    except exceptions.NotFoundException:
      pass
    self._logger.warning('Failed to get cached user by name. Attempting to '
                         'refresh cache.')
    users, groups = self._api_client.get_users_and_groups(name)
    self._cache.repopulate_users_and_groups(users, groups)
    return [self._user_to_passwd_line(self._cache.get_user_by_name(name))]

  def _get_user_by_uid(self, uid):
    self._logger.info('Getting user by uid: [%d]', uid)
    return [self._user_to_passwd_line(self._cache.get_user_by_uid(uid))]

  def _get_users(self):
    self._logger.info('Getting users.')
    return [self._user_to_passwd_line(user) for user in self._cache.get_users()]

  def _get_group_by_name(self, name):
    self._logger.info('Getting group by name: [%s]', name)
    return [self._group_to_group_line(self._cache.get_group_by_name(name))]

  def _get_group_by_gid(self, gid):
    self._logger.info('Getting group by gid: [%d]', gid)
    return [self._group_to_group_line(self._cache.get_group_by_gid(gid))]

  def _get_groups(self):
    self._logger.info('Getting groups.')
    return [self._group_to_group_line(group)
            for group in self._cache.get_groups()]

  def _get_account_names(self):
    self._logger.info('Getting account names.')
    user_names = set([user.name for user in self._cache.get_users()])
    group_names = set([group.name for group in self._cache.get_groups()])
    return list(user_names.union(group_names))

  def _is_account_name(self, name):
    self._logger.info('Validating account name: [%s]', name)
    self._cache.validate_account_name(name)

  def _get_authorized_keys(self, user_name):
    """Returns authorized keys for a user. Falls back to cache on error."""
    self._logger.info('Getting authorized keys: [%s]', user_name)
    try:
      keys = self._api_client.get_authorized_keys(user_name)
      self._cache.cache_authorized_keys(user_name, keys)
      return keys.authorized_keys
    except (exceptions.BackendException, exceptions.OutOfQuotaException) as e:
      self._logger.exception('Failed to fetch authorized keys. Falling back to '
                             'cache.')
      original_exception = e
    try:
      keys = self._cache.get_authorized_keys(user_name)
    except exceptions.NotFoundException:
      raise original_exception
    return keys.authorized_keys

  def _handle_impl(self):
    """Handles a request send to the AccountsProxy server."""
    start_sec = time.time()
    self._socket.settimeout(_SOCKET_TIMEOUT_SEC)
    try:
      command = self._socket.recv(128)
      self._logger.debug('Command received: [%s]', command)
      method, _, arg = command.partition(' ')
      handler, arg_parser = {
          'get_user_by_name': (self._get_user_by_name, lambda: (arg,)),
          'get_user_by_uid': (self._get_user_by_uid, lambda: (int(arg),)),
          'get_users': (self._get_users, lambda: ()),
          'get_group_by_name': (self._get_group_by_name, lambda: (arg,)),
          'get_group_by_gid': (self._get_group_by_gid, lambda: (int(arg),)),
          'get_groups': (self._get_groups, lambda: ()),
          'get_account_names': (self._get_account_names, lambda: ()),
          'is_account_name': (self._is_account_name, lambda: (arg,)),
          'get_authorized_keys': (self._get_authorized_keys, lambda: (arg,))
      }[method]
      arg_tuple = arg_parser()
    except socket.error:
      self._logger.exception('Error while reading command.')
      self._write_response('400')
      return
    except (KeyError, ValueError):
      self._logger.exception('Invalid command received: [%s]', command)
      self._write_response('400')
      return

    try:
      info_lines = handler(*arg_tuple)
      self._logger.info('Request succeeded.')
      output_lines = ['200']
      output_lines.extend(info_lines or [])
      self._logger.debug('Output: [%s]', output_lines)
      self._write_response('\n'.join(output_lines))
    except exceptions.NotFoundException:
      self._logger.exception('Requested user or group does not exist.')
      self._write_response('404')
    except exceptions.LookupException:
      self._logger.exception('Failed to send accounts request.')
      self._write_response('500')

    self._logger.debug('Request duration: [%f sec]', time.time() - start_sec)

  def _write_response(self, data):
    try:
      self._socket.sendall(data)
    except socket.error:
      self._logger.exception('Error while attempting to write to socket.')

  def handle(self):
    try:
      self._handle_impl()
    except BaseException as e:
      self._logger.exception('Unrecoverable error during request handling.')
      self._exception_handler(e)
