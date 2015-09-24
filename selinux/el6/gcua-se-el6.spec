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

Name:           gcua-se-el6
Version:        0.1
Release:        0%{?dist}
Summary:        SELinux policy module for gcua
Group:          System Environment/Base
Distribution:   gcua
License:        Apache 2.0
URL:            https://cloud.google.com/compute/docs/access/user-accounts/
Vendor:         Google, Inc
Packager:       Derek Olds <olds@google.com>

Source1:	gcua-se-el6.pp

%define relabel_files() \
restorecon -R /usr/share/google/gcua; \

%define _topdir %(pwd)
%define WORKINGDIR %_topdir/selinux/el6
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
This package installs and sets up the  SELinux policy security module for gcua.

%build
make -f /usr/share/selinux/devel/Makefile

%install
install -d %{buildroot}%{_datadir}/selinux/packages
install -m 644 %{SOURCE1} %{buildroot}%{_datadir}/selinux/packages

%clean
mkdir -p %_topdir/pkg
mv noarch/* %_topdir/pkg/
rm -rf tmp/ noarch/ .build/ *.pp

%post
semodule -i %{_datadir}/selinux/packages/gcua-se-el6.pp
%relabel_files
/etc/init.d/gcua restart
exit 0

%postun
if [ $1 -eq 0 ]; then
    semodule -r gcua
    %relabel_files
fi;
exit 0

%files
%attr(0600,root,root) %{_datadir}/selinux/packages/gcua-se-el6.pp


%changelog
* Thu Sep 10 2015 Derek Olds olds@google.com 0.0-3
- testing relocatable builds

