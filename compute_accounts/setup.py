# Copyright 2015 Google Inc. All Rights Reserved.
"""Package setup file."""

from setuptools import setup

setup(name='compute_accounts',
      version='0.1',
      description='Tools for communicating with the Compute Accounts service.',
      url='',
      author='',
      author_email='',
      license='Apache 2.0',
      packages=['compute_accounts'],
      scripts=[
          'bin/authorized_keys.py',
          'bin/proxy_daemon.py'
      ],
      install_requires=[
          'jsonschema',
          'lockfile',
          'python-daemon',
          'requests',
          'simplejson'
      ],
      test_suite='nose.collector',
      tests_require=[
          'mock',
          'nose',
          'responses',
          'tl.testing'
      ],
      zip_safe=False)
