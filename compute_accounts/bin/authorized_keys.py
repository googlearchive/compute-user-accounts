#!/usr/bin/python
# Copyright 2015 Google Inc. All Rights Reserved.
"""Gets authorized keys for a user and prints them to stdout."""

import sys
import compute_accounts


def main():
  assert len(sys.argv) == 2
  command = 'get_authorized_keys ' + sys.argv[1]
  # This always send a request to the API, so extend the timeout.
  for line in compute_accounts.get_account_info(command, timeout=5):
    sys.stdout.write(line)
    sys.stdout.write('\n')


if __name__ == '__main__':
  main()
