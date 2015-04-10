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

"""Exceptions raised by the compute_accounts module."""


class LookupException(Exception):
  """Base class for exceptions raised during Compute Accounts lookups."""

  def __init__(self, format_string, *args, **kwargs):
    super(LookupException, self).__init__()
    self.message = format_string.format(*args, **kwargs)

  def __str__(self):
    return self.message


class BackendException(LookupException):
  """An error occured while sending request to the Compute Accounts service."""

  def __init__(self, inner_exception, format_string, *args, **kwargs):
    super(BackendException, self).__init__(format_string, *args, **kwargs)
    self.message += '\n{}: {}'.format(type(inner_exception).__name__,
                                      inner_exception)
    self.inner_exception = inner_exception


class NotFoundException(LookupException):
  """The requested view does not exist."""


class OutOfQuotaException(LookupException):
  """There is no quota available to send the request."""
