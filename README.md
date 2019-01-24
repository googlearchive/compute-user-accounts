# Status: Archived
This repository has been archived and is no longer maintained.

![status: inactive](https://img.shields.io/badge/status-inactive-red.svg)


Google Compute User Accounts - VM Software
===

Give yourself and others access to your virtual machines by creating Linux user
accounts. User accounts have their own username and home directory on your
virtual machine instances and can be created and managed programmatically
through an API. Once you have created a user account, it can be used to log in
to the Linux virtual machines in your project.

Installation
---

0. [Install Go](https://golang.org/doc/install)
0. [Install FPM](https://github.com/jordansissel/fpm#get-with-the-download)
0. Make a deb/rpm: `make package`
0. Install the deb/rpm on your VM: `dpkg -i pkg/gcua_X.XXXXXXXX.XXXXXX_amd64.deb` or `rpm -ivh pkg/gcua-X.XXXXXXXX.XXXXXX-X.x86_64.rpm`

Usage
---

See [GCP Documentation](https://cloud.google.com/compute/docs/access/user-accounts/)

Contributing
---

See [Contributing](CONTRIBUTING.md)
