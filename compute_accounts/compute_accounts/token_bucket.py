# Copyright 2015 Google Inc. All Rights Reserved.
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
