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

ACCOUNT=gcua
DIR=/usr/share/google

# Add user account and script directory.
if ! grep -q ${ACCOUNT} /etc/passwd; then
  # Force useradd to start allocating local users in the 50000 range.
  useradd --system --user-group --uid 50000 --comment "UID is 50000 to avoid UID collisions for new local users" ${ACCOUNT}
fi

# This is required by sshd for AuthorizedKeyCommand.
chmod -R 755 ${DIR}

# This is required by sudo.
chown root:root /etc/sudoers.d/gcua
chmod 440 /etc/sudoers.d/gcua

# Install the NSS plugin.
chmod 0644 /usr/lib/libnss_google.so.2.0.1
ldconfig

# (Re)Start daemon.
update-rc.d gcua defaults
service gcua restart

# Enable lazy home directory creation.
if ! grep -q pam_mkhomedir.so /etc/pam.d/sshd; then
  echo "session     required      pam_mkhomedir.so skel=/etc/skel umask=0022" >> /etc/pam.d/sshd
fi

# Enable NSS plugin.
if ! grep -q google /etc/nsswitch.conf; then
  sed -i -r "s/^((passwd|group|shadow):\s+compat)/\1 google/" /etc/nsswitch.conf
fi

# Enable AuthorizedKeysCommand and restart sshd.
if ! grep -q ${DIR}/authorizedkeys /etc/ssh/sshd_config; then
  echo "AuthorizedKeysCommand ${DIR}/authorizedkeys" >> /etc/ssh/sshd_config
  echo "AuthorizedKeysCommandUser ${ACCOUNT}" >> /etc/ssh/sshd_config
  service ssh restart
fi
