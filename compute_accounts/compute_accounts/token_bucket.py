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

"""A token bucket used to rate limit requests."""

import threading
import time

from . import exceptions


class TokenBucket(object):
  """A token bucket used to limit maximum amortized/burst request rates.

  This class this thread-safe.
  """

  def __init__(self, bucket_size, token_creation_sec):
    """Inits the token bucket with a size and a token creation rate."""
    assert bucket_size >= 1
    assert token_creation_sec > 0

    self._lock = threading.Lock()
    self._capacity = float(bucket_size)
    self._fill_rate_per_sec = 1.0 / float(token_creation_sec)
    self._current_level = self._capacity
    self._last_fill_time = time.time()

  def consume(self):
    """Consumes a token from the bucket.

    Raises:
      OutOfQuotaException: No tokens are available to consume.
    """
    with self._lock:
      self._fill_bucket()
      if self._current_level < 1:
        seconds_to_token = (1 - self._current_level) / self._fill_rate_per_sec
        raise exceptions.OutOfQuotaException(
            'No quota available for {} seconds.', seconds_to_token)
      self._current_level -= 1

  def _fill_bucket(self):
    """Fills the token bucket with tokens created since the last fill."""
    now = time.time()
    delta_sec = now - self._last_fill_time
    if delta_sec > 0:  # Otherwise there was clock skew.
      new_level = self._current_level + (self._fill_rate_per_sec * delta_sec)
      self._current_level = min(new_level, self._capacity)
    self._last_fill_time = now
