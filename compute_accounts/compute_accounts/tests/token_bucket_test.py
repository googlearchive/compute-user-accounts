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

"""Tests for TokenBucket."""

import unittest

import compute_accounts
import mock


class TocketBucketTest(unittest.TestCase):

  @mock.patch('compute_accounts.token_bucket.time.time')
  def test_basic_case(self, time_mock):
    time_mock.return_value = 0.0
    # Two token capacity, three seconds to fill.
    bucket = compute_accounts.token_bucket.TokenBucket(2, 3)
    # Two tokens in bucket.
    bucket.consume()
    bucket.consume()
    # Zero tokens in bucket.
    with self.assertRaisesRegexp(compute_accounts.OutOfQuotaException,
                                 'No quota available for 3.0 seconds.'):
      bucket.consume()
    time_mock.return_value = 3.0
    # One token in bucket (created because 3 seconds passed).
    bucket.consume()
    time_mock.return_value = 5.9
    # Fraction of token in bucket.
    with self.assertRaisesRegexp(compute_accounts.OutOfQuotaException,
                                 'No quota available for 0.1 seconds.'):
      bucket.consume()
    time_mock.return_value = 6.0
    # One token in bucket.
    bucket.consume()
    time_mock.return_value = 7.0
    # Zero tokens in bucket.
    with self.assertRaisesRegexp(compute_accounts.OutOfQuotaException,
                                 'No quota available for 2.0 seconds.'):
      bucket.consume()

  @mock.patch('compute_accounts.token_bucket.time.time')
  def test_bucket_does_not_overflow(self, time_mock):
    time_mock.return_value = 0.0
    # One token capacity, one second to fill.
    bucket = compute_accounts.token_bucket.TokenBucket(1, 1)
    bucket.consume()
    with self.assertRaisesRegexp(compute_accounts.OutOfQuotaException,
                                 'No quota available for 1.0 seconds.'):
      bucket.consume()
    time_mock.return_value = 10000.0
    bucket.consume()
    with self.assertRaisesRegexp(compute_accounts.OutOfQuotaException,
                                 'No quota available for 1.0 seconds.'):
      bucket.consume()

  @mock.patch('compute_accounts.token_bucket.time.time')
  def test_bucket_handles_clock_skew(self, time_mock):
    time_mock.return_value = 10.0
    # One token capacity, one second to fill.
    bucket = compute_accounts.token_bucket.TokenBucket(1, 1)
    bucket.consume()
    with self.assertRaisesRegexp(compute_accounts.OutOfQuotaException,
                                 'No quota available for 1.0 seconds.'):
      bucket.consume()
    time_mock.return_value = 0.0
    # Zero tokens in bucket.
    with self.assertRaisesRegexp(compute_accounts.OutOfQuotaException,
                                 'No quota available for 1.0 seconds.'):
      bucket.consume()
    time_mock.return_value = 1.0
    bucket.consume()
    with self.assertRaisesRegexp(compute_accounts.OutOfQuotaException,
                                 'No quota available for 1.0 seconds.'):
      bucket.consume()

  def test_invalid_parameters(self):
    with self.assertRaises(AssertionError):
      compute_accounts.token_bucket.TokenBucket(0, 1)
    with self.assertRaises(AssertionError):
      compute_accounts.token_bucket.TokenBucket(.9, 1)
    with self.assertRaises(AssertionError):
      compute_accounts.token_bucket.TokenBucket(1, 0)
