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

# Stop daemon.
if [ -x /bin/systemctl ]; then
  # Systemd.
  systemctl --no-reload disable gcua
  systemctl stop gcua
elif [ -x /sbin/chkconfig ]; then
  # System-V on RHEL.
  chkconfig --del gcua
  service gcua stop
else
  # System-V on Debian.
  update-rc.d gcua remove
  service gcua stop
fi

# Disable NSS plugin.
sed -i "s/ google//" /etc/nsswitch.conf

# Disable AuthorizedKeysCommand.
sed -i "s#AuthorizedKeysCommand ${DIR}/authorizedkeys##" /etc/ssh/sshd_config
sed -i "s/AuthorizedKeysCommandUser ${ACCOUNT}//" /etc/ssh/sshd_config
sed -i "s/AuthorizedKeysCommandRunAs ${ACCOUNT}//" /etc/ssh/sshd_config

# Restart sshd.
if [ -x /bin/systemctl ]; then
  # Systemd.
  systemctl reload sshd
elif [ -x /sbin/chkconfig ]; then
  # System-V on RHEL.
  service sshd restart
else
  # System-V on Debian.
  service ssh restart
fi
