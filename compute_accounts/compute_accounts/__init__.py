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
