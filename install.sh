#!/bin/bash
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

set -o pipefail
set -e

if [ "$(id -u)" != "0" ]; then
  echo "This script must be run as root." 1>&2
  exit 1
fi

ACCOUNT="compute_accounts"
SCRIPT_DIR="/usr/share/google/compute_accounts"
INSTALL_APT_DEPS=true
START_DAEMON=true
RESTART_SSHD=true
DAEMON_ARGS=""

# Add user account and script directory.
if ! grep -q ${ACCOUNT} /etc/passwd; then
  # Force useradd to start allocating local users in the 50000 range.
  useradd --system --user-group --uid 50000 --comment "UID is 50000 to avoid UID collisions for new local users" ${ACCOUNT}
fi
if [ ! -d "${SCRIPT_DIR}" ]; then
  mkdir -p ${SCRIPT_DIR}
fi

# Get necessary packages.
if [ "${INSTALL_APT_DEPS}" = true ]; then
  sudo apt-get update
  sudo apt-get install -y python-pip g++ make
fi

# Install the daemon and authorized keys command.
pushd compute_accounts
python setup.py install --install-scripts=${SCRIPT_DIR}
popd
# This is required by sshd for AuthorizedKeyCommand.
chmod -R 755 ${SCRIPT_DIR}
# This is required by sudo.
chmod 440 /etc/sudoers.d/compute-accounts

# Build and install the nss plugin.
make -C nss_plugin release
install -m 0644 nss_plugin/bin/libnss_google.so.2.0.1 /usr/lib/x86_64-linux-gnu/
ldconfig

# Register init.d script.
update-rc.d compute-accounts defaults

# (Re)Start daemon.
if [ "${START_DAEMON}" = true ]; then
  /etc/init.d/compute-accounts restart ${DAEMON_ARGS}
fi

# Enable lazy home directory creation.
if ! grep -q pam_mkhomedir.so /etc/pam.d/sshd; then
  echo "session     required      pam_mkhomedir.so skel=/etc/skel umask=0022" >> /etc/pam.d/sshd
fi

# Enable nss plugin.
if ! grep -q google /etc/nsswitch.conf; then
  sed -i -r "s/^((passwd|group|shadow):\s+compat)/\1 google/" /etc/nsswitch.conf
fi

# Enable authorized keys command and restart sshd.
if ! grep -q ${SCRIPT_DIR}/authorized_keys.py /etc/ssh/sshd_config; then
  echo "AuthorizedKeysCommand ${SCRIPT_DIR}/authorized_keys.py" >> /etc/ssh/sshd_config
  echo "AuthorizedKeysCommandUser ${ACCOUNT}" >> /etc/ssh/sshd_config
fi
if [ "${RESTART_SSHD}" = true ]; then
  /etc/init.d/ssh restart
fi
