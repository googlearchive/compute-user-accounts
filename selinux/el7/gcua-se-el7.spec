# vim: sw=4:ts=4:et
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

Name:   gcua-se-el7
Version:        0.1
Release:        0%{?dist}
Summary:        SELinux policy module for gcua-se-el7
Group:          System Environment/Base
Distribution:   gcua
License:        Apache 2.0
URL:            https://cloud.google.com/compute/docs/access/user-accounts/
Vendor:         Google, Inc
Packager:       Derek Olds <olds@google.com>

Source0:	gcua-se-el7.pp

%define relabel_files() \
restorecon -R /usr/share/google/gcua; \
restorecon -R /etc/systemd/system/gcua.service; \

%define selinux_policyver 3.13.1-23

%define _topdir %(pwd)
%define WORKINGDIR %_topdir/selinux/el7
%define _sourcedir %WORKINGDIR
%define _specdir %WORKINGDIR
%define _srcrpmdir %WORKINGDIR
%define _rpmdir %WORKINGDIR
%define _builddir %WORKINGDIR
%define _buildrootdir %WORKINGDIR/.build


BuildRequires: selinux-policy-devel
Requires: policycoreutils, libselinux-utils
Requires(post): selinux-policy-base >= %{selinux_policyver}, policycoreutils
Requires(postun): policycoreutils
Requires(post): gcua
BuildArch: noarch

%description
This package installs and sets up the SELinux policy security module for gcua on el7.


%build
make -f /usr/share/selinux/devel/Makefile

%install
install -d %{buildroot}%{_datadir}/selinux/packages
install -m 644 %{SOURCE0} %{buildroot}%{_datadir}/selinux/packages
install -d %{buildroot}/etc/selinux/targeted/contexts/users/

%clean
mkdir -p %_topdir/pkg
mv noarch/* %_topdir/pkg/
rm -rf tmp/ noarch/ .build/ *.pp

%post
semodule -n -i %{_datadir}/selinux/packages/gcua-se-el7.pp
if /usr/sbin/selinuxenabled ; then
    /usr/sbin/load_policy
    %relabel_files

fi;
exit 0

%postun
if [ $1 -eq 0 ]; then
    semodule -n -r gcua-se-el7
    if /usr/sbin/selinuxenabled ; then
       /usr/sbin/load_policy
       %relabel_files

    fi;
fi;
exit 0

%files
%attr(0600,root,root) %{_datadir}/selinux/packages/gcua-se-el7.pp


%changelog
* Tue Sep 22 2015 YOUR NAME <YOUR@EMAILADDRESS> 1.0-1
- Initial version

