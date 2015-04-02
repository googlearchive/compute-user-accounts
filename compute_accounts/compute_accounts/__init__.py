# Copyright 2015 Google Inc. All Rights Reserved.
"""Package marker file for the compute_accounts package."""

from .exceptions import BackendException
from .exceptions import LookupException
from .exceptions import NotFoundException
from .exceptions import OutOfQuotaException
import logging

from .accounts_proxy import AccountsProxy
from .accounts_proxy import UNIX_SOCKET_PATH
from .proxy_client import get_account_info

try:
  # Available as of Python 2.7
  # pylint:disable=g-import-not-at-top
  from logging import NullHandler
  # pylint:enable=g-import-not-at-top
except ImportError:

  class NullHandler(logging.Handler):

    def emit(self, record):
      pass

logging.getLogger(__name__).addHandler(NullHandler())
del NullHandler
