#!/bin/bash

if [ "$(id -u)" != "0" ]; then
  echo "This script must be run as root." 1>&2
  exit 1
fi

OPTIND=1

flag_count=0
daemon_args=""

while getopts "hd:s" opt; do
  case "$opt" in
  h|\?)
    echo "sudo ./install [-s | -d <CORP_IP> ]" 1>&2
    exit 1
    ;;
  d)
    daemon_args="--api-root http://${OPTARG}:3990/ --logging-level debug"
    flag_count=$((flag_count + 1))
    ;;
  s)
    daemon_args="--computeaccounts-api-version stagingvm_alpha
      --compute-api-version staging_v1
      --logging-level debug"
    flag_count=$((flag_count + 1))
    ;;
  esac
done

if [ "${flag_count}" -gt 1 ]; then
  echo "-s and -d are mutually exclusive." 1>&2
  exit 1
fi

ACCOUNT=compute_accounts
RUN_DIR=/var/run/compute_accounts
LOG_DIR=/var/log/compute_accounts
SCRIPT_DIR=/usr/share/google/compute_accounts

set -o pipefail
set -e

# Add user and system directories.
if ! grep -q ${ACCOUNT} /etc/passwd; then
  # Force useradd to start allocating local users in the 50000 range.
  useradd --system --user-group --uid 50000 --comment "UID is 50000 to avoid UID collisions for new local users" ${ACCOUNT}
fi
if [ ! -d "${RUN_DIR}" ]; then
  mkdir ${RUN_DIR}
  chown ${ACCOUNT}:${ACCOUNT} ${RUN_DIR}
fi
if [ ! -d "${LOG_DIR}" ]; then
  mkdir ${LOG_DIR}
  chown ${ACCOUNT}:${ACCOUNT} ${LOG_DIR}
  chmod 750 ${LOG_DIR}
fi
if [ ! -d "${SCRIPT_DIR}" ]; then
  mkdir -p ${SCRIPT_DIR}
  # This is required by sshd for AuthorizedKeyCommand.
  chmod 755 ${SCRIPT_DIR}
fi

# Get necessary packages.
sudo apt-get update
sudo apt-get install -y python-pip g++ make

# Install the daemon and authorized keys command.
pushd compute_accounts
python setup.py install --install-scripts=${SCRIPT_DIR}
popd
# This is required by sshd for AuthorizedKeyCommand.
chmod 755 ${SCRIPT_DIR}/authorized_keys.py

# Build and install the nss plugin.
make -C nss_plugin release
install -m 0644 nss_plugin/bin/libnss_google.so.2.0.1 /usr/lib/x86_64-linux-gnu/
ldconfig

# (Re)Start daemon.
sudo pkill -u ${ACCOUNT} || true
sudo ${SCRIPT_DIR}/proxy_daemon.py ${daemon_args}

# Enable sudo.
if ! grep -q gce-sudoers /etc/sudoers; then
  echo "%gce-sudoers ALL=(ALL:ALL) NOPASSWD:ALL" >> /etc/sudoers
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
/etc/init.d/ssh restart
