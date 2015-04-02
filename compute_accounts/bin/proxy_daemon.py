#!/usr/bin/python
# Copyright 2015 Google Inc. All Rights Reserved.
"""Launches the Compute Accounts proxy server as a daemon."""

import argparse
import logging
import logging.handlers
import pwd

import compute_accounts
import daemon
import lockfile

_DAEMON_ACCOUNT = 'compute_accounts'
_LOGGER_LOG_PATH = '/var/log/compute_accounts/daemon.log'
_STDOUT_LOG_PATH = '/var/log/compute_accounts/daemon.out'
_STDERR_LOG_PATH = '/var/log/compute_accounts/daemon.err'
_PID_FILE_PATH = '/var/run/compute_accounts/pid'


def _parse_args():
  """Returns arguments parsed by argparse."""
  parser = argparse.ArgumentParser(
      description='Launch the compute_accounts daemon.')
  parser.add_argument(
      '--logging-level', default='INFO', type=str.upper,
      choices=['DEBUG', 'INFO', 'WARNING', 'ERROR', 'CRITICAL'])
  parser.add_argument(
      '--api-root', default='https://www.googleapis.com/')
  parser.add_argument(
      '--computeaccounts-api-version', default='alpha')
  parser.add_argument(
      '--compute-api-version', default='v1')
  args = parser.parse_args()

  return args


def _start_daemon(args):
  """Starts the Compute Accounts daemon and servers traffic forever."""
  logger = logging.getLogger()
  logger.setLevel(getattr(logging, args.logging_level))
  handler = logging.handlers.RotatingFileHandler(
      _LOGGER_LOG_PATH, maxBytes=1048576, backupCount=10)
  formatter = logging.Formatter('[%(asctime)s] [%(levelname)-8s] '
                                '(%(filename)s:%(lineno)s) --- %(message)s')
  handler.setFormatter(formatter)
  logger.addHandler(handler)

  try:
    logger.info('Starting Compute Accounts daemon.')
    compute_accounts.AccountsProxy(
        args.api_root, args.computeaccounts_api_version,
        args.compute_api_version).start()
  except:
    logger.exception('Daemon failed.')
    raise
  finally:
    logger.info('Daemon shutting down.')
    logging.shutdown()


def start():
  """Starts the Compute Accounts daemon with daemonizing."""
  args = _parse_args()
  pidfile = lockfile.FileLock(_PID_FILE_PATH)
  daemon_account = pwd.getpwnam(_DAEMON_ACCOUNT)
  stdout = open(_STDOUT_LOG_PATH, 'w+')
  stderr = open(_STDERR_LOG_PATH, 'w+')
  with daemon.DaemonContext(
      pidfile=pidfile, stdout=stdout, stderr=stderr, uid=daemon_account.pw_uid,
      gid=daemon_account.pw_gid):
    _start_daemon(args)


def start_without_daemonizing():
  """Starts the Compute Accounts daemon without daemonizing."""
  args = _parse_args()
  _start_daemon(args)


if __name__ == '__main__':
  start()
