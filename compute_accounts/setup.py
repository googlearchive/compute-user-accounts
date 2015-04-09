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
      data_files=[('/etc/init.d', ['etc/init.d/compute-accounts']),
                  ('/etc/sudoers.d', ['etc/sudoers.d/compute-accounts'])],
      install_requires=[
          'jsonschema',
          'pidfile',
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
