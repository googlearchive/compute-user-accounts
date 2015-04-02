# Copyright 2015 Google Inc. All Rights Reserved.
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
