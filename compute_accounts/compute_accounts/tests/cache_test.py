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

"""Tests for Cache."""

import unittest

import compute_accounts
from compute_accounts.cache import Cache
from compute_accounts.entities import AuthorizedKeys
from compute_accounts.entities import Group
from compute_accounts.entities import User
import mock


class CacheTest(unittest.TestCase):

  def test_cache_basic_case(self):
    cache = Cache()
    user1 = User(name='user1', uid=1001, gid=1001, gecos='', dir='/home/user1',
                 shell='/bin/bash')
    user2 = User(name='user2', uid=1002, gid=1001, gecos='', dir='/home/user2',
                 shell='/bin/bash')
    group1 = Group(name='group1', gid=1001, mem=('user1', 'user2'))
    group2 = Group(name='group2', gid=1002, mem=())
    cache.repopulate_users_and_groups((user1, user2), (group1, group2))
    cache.validate_account_name('user1')
    cache.validate_account_name('group2')
    self.assertEqual(user1, cache.get_user_by_name('user1'))
    self.assertEqual(user2, cache.get_user_by_uid(1002))
    self.assertEqual(group1, cache.get_group_by_name('group1'))
    self.assertEqual(group2, cache.get_group_by_gid(1002))
    self.assertItemsEqual((user1, user2), cache.get_users())
    self.assertItemsEqual((group1, group2), cache.get_groups())
    with self.assertRaisesRegexp(compute_accounts.NotFoundException,
                                 r'Not found in cache: \[nonexistent-user\]'):
      cache.get_user_by_name('nonexistent-user')
    with self.assertRaisesRegexp(compute_accounts.NotFoundException,
                                 r'Not found in cache: \[123\]'):
      cache.get_user_by_uid(123)
    with self.assertRaisesRegexp(compute_accounts.NotFoundException,
                                 r'Not found in cache: \[nonexistent-group\]'):
      cache.get_group_by_name('nonexistent-group')
    with self.assertRaisesRegexp(compute_accounts.NotFoundException,
                                 r'Not found in cache: \[123\]'):
      cache.get_group_by_gid(123)
    with self.assertRaisesRegexp(compute_accounts.NotFoundException,
                                 r'Not found in cache: \[nonexistent-group\]'):
      cache.validate_account_name('nonexistent-group')

  @mock.patch('compute_accounts.token_bucket.time.time', new=lambda: 123.2)
  def test_cache_authorized_keys(self):
    keys = AuthorizedKeys(timestamp=123.1, authorized_keys=(
        'ssh-rsa AAAAB3NzaC1y2EAAAADAQABAAABAQDVTRZ9YV user1@gmail.com',
        'ssh-rsa AAAAB3NzaCDVTRZ9YV user2@gmail.com'))
    cache = Cache()
    cache.cache_authorized_keys('user1', keys)
    self.assertEqual(keys, cache.get_authorized_keys('user1'))

  @mock.patch('compute_accounts.token_bucket.time.time', new=lambda: 60 * 30.0)
  def test_cache_authorized_keys_stale(self):
    keys = AuthorizedKeys(timestamp=0.0, authorized_keys=(
        'ssh-rsa AAAAB3NzaC1y2EAAAADAQABAAABAQDVTRZ9YV user1@gmail.com',
        'ssh-rsa AAAAB3NzaCDVTRZ9YV user2@gmail.com'))
    cache = Cache()
    cache.cache_authorized_keys('user1', keys)
    with self.assertRaisesRegexp(compute_accounts.NotFoundException,
                                 r'Cached user keys are stale: \[user1\]'):
      cache.get_authorized_keys('user1')

  @mock.patch('compute_accounts.token_bucket.time.time', new=lambda: 1.0)
  def test_cache_authorized_keys_removed_for_removed_user(self):
    user = User(name='user1', uid=1001, gid=1001, gecos='', dir='/home/user1',
                shell='/bin/bash')
    keys = AuthorizedKeys(timestamp=1.0, authorized_keys=(
        'ssh-rsa AAAAB3NzaC1y2EAAAADAQABAAABAQDVTRZ9YV user1@gmail.com',
        'ssh-rsa AAAAB3NzaCDVTRZ9YV user2@gmail.com'))
    cache = Cache()
    cache.cache_authorized_keys('user1', keys)
    self.assertEqual(keys, cache.get_authorized_keys('user1'))
    cache.repopulate_users_and_groups((user,), ())
    self.assertEqual(keys, cache.get_authorized_keys('user1'))
    cache.repopulate_users_and_groups((), ())
    with self.assertRaisesRegexp(compute_accounts.NotFoundException,
                                 r'Not found in cache: \[user1\]'):
      cache.get_authorized_keys('user1')

  @mock.patch('compute_accounts.token_bucket.time.time', new=lambda: 60 * 30.0)
  def test_cache_stale_authorized_keys_removed_during_refresh(self):
    user = User(name='user1', uid=1001, gid=1001, gecos='', dir='/home/user1',
                shell='/bin/bash')
    keys = AuthorizedKeys(timestamp=0.0, authorized_keys=(
        'ssh-rsa AAAAB3NzaC1y2EAAAADAQABAAABAQDVTRZ9YV user1@gmail.com',
        'ssh-rsa AAAAB3NzaCDVTRZ9YV user2@gmail.com'))
    cache = Cache()
    cache.cache_authorized_keys('user1', keys)
    with self.assertRaisesRegexp(compute_accounts.NotFoundException,
                                 r'Cached user keys are stale: \[user1\]'):
      cache.get_authorized_keys('user1')
    cache.repopulate_users_and_groups((user,), ())
    with self.assertRaisesRegexp(compute_accounts.NotFoundException,
                                 r'Not found in cache: \[user1\]'):
      cache.get_authorized_keys('user1')


if __name__ == '__main__':
  unittest.main()
