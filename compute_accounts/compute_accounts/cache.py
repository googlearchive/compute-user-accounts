# Copyright 2015 Google Inc. All Rights Reserved.
"""Cache for storing Compute Accounts information in memory."""

import logging
import threading
import time

from . import exceptions

_KEY_ENTRY_EXPIRATION_SEC = 60 * 30  # 30 minutes.


class Cache(object):
  """Cache for storing Compute Accounts information in memory.

  This class is thread-safe.
  """

  def __init__(self):
    """Inits the Cache."""
    self._logger = logging.getLogger(__name__)
    self._lock = threading.Lock()
    self._users_by_name = {}
    self._users_by_uid = {}
    self._groups_by_name = {}
    self._groups_by_gid = {}
    self._keys_by_user_name = {}

  def _get_value(self, cache_data, key):
    with self._lock:
      result = cache_data.get(key)
    if not result:
      raise exceptions.NotFoundException('Not found in cache: [{}]', key)
    return result

  def _get_values(self, cache_data):
    with self._lock:
      return tuple(cache_data.values())

  def _is_key_entry_fresh(self, keys):
    delta_sec = time.time() - keys.timestamp
    return delta_sec >= 0 and delta_sec < _KEY_ENTRY_EXPIRATION_SEC

  def validate_account_name(self, name):
    """Raises a NotFoundException if name is not a valid user or group name."""
    with self._lock:
      if name not in self._users_by_name and name not in self._groups_by_name:
        raise exceptions.NotFoundException('Not found in cache: [{}]', name)

  def get_user_by_name(self, name):
    """Gets a cached user entry by name."""
    self._logger.debug('Getting user from cache by name: [%s]', name)
    return self._get_value(self._users_by_name, name)

  def get_user_by_uid(self, uid):
    """Gets a cached user entry by uid."""
    self._logger.debug('Getting user from cache by uid: [%d]', uid)
    return self._get_value(self._users_by_uid, uid)

  def get_users(self):
    """Gets all cached user entries."""
    self._logger.debug('Getting users from cache.')
    return self._get_values(self._users_by_name)

  def get_group_by_name(self, name):
    """Gets a cached group entry by name."""
    self._logger.debug('Getting group from cache by name: [%s]', name)
    return self._get_value(self._groups_by_name, name)

  def get_group_by_gid(self, gid):
    """Gets a cached group entry by gid."""
    self._logger.debug('Getting group from cache by gid: [%d]', gid)
    return self._get_value(self._groups_by_gid, gid)

  def get_groups(self):
    """Gets all cached group entries."""
    self._logger.debug('Getting groups from cache.')
    return self._get_values(self._groups_by_name)

  def get_authorized_keys(self, user_name):
    """Gets a user's cached authorized keys as long as they are not expired."""
    self._logger.debug('Getting authorized keys from cache: [%s]', user_name)
    keys = self._get_value(self._keys_by_user_name, user_name)
    if not self._is_key_entry_fresh(keys):
      raise exceptions.NotFoundException('Cached user keys are stale: [{}]',
                                         user_name)
    self._logger.debug('Keys are fresh.')
    return keys

  def repopulate_users_and_groups(self, users, groups):
    """Repopulate cached users and groups. Remove invalid/stale keys."""
    self._logger.debug('Repopulating cache from fetched users and groups.')
    with self._lock:
      self._users_by_name.clear()
      self._users_by_uid.clear()
      valid_user_names = set()
      # Repopulate user dicts while saving the set of valid user names.
      for user in users:
        self._users_by_name[user.name] = user
        self._users_by_uid[user.uid] = user
        valid_user_names.add(user.name)
      self._groups_by_name.clear()
      self._groups_by_gid.clear()
      # Repopulate group dicts.
      for group in groups:
        self._groups_by_name[group.name] = group
        self._groups_by_gid[group.gid] = group
      new_keys = {}
      # Filter out cached keys that are either expired or do not belong to a
      # valid user.
      for user_name in self._keys_by_user_name:
        keys = self._keys_by_user_name[user_name]
        if user_name in valid_user_names and self._is_key_entry_fresh(keys):
          new_keys[user_name] = keys
      self._keys_by_user_name = new_keys

  def cache_authorized_keys(self, user_name, authorized_keys):
    """Cache a user's authorized keys."""
    self._logger.debug('Populating cache from fetched authorized keys.')
    with self._lock:
      self._keys_by_user_name[user_name] = authorized_keys
