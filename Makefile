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
.PHONY: all proto build debug test cover package clean

VERSION:=0.$(shell date -u +%Y%m%d.%H%M%S)
PACKAGE_NAME:="gcua"
DESCRIPTION:="Google Compute User Accounts."
LICENSE:="Apache 2.0"
MAINTAINER:="gc-team@google.com"
VENDOR:="Google Inc."
URL:="https://github.com/GoogleCloudPlatform/compute-user-accounts"
LIBC_VERSION=2.12
LIBSTDCXX_VERSION=4.4.7

SOCKET_PATH:=/var/run/gcua.socket
BTARGET:=build
GOFLAGS=-ldflags "-X main.version ${VERSION} -X github.com/GoogleCloudPlatform/compute-user-accounts/server.socketPath ${SOCKET_PATH}"
TTARGET:=test
TFLAGS:=

SOURCES:= \
  ${GOPATH}/bin/gcua=/usr/share/google/ \
  ${GOPATH}/bin/authorizedkeys=/usr/share/google/ \
  nssplugin/bin/libnss_google.so.2.0.1=/usr/lib/ \
  etc/init.d/gcua \
  etc/sudoers.d/gcua \
  etc/systemd/system/gcua.service

FPM_ARGS:= \
  -s dir -n ${PACKAGE_NAME} -v ${VERSION} -a native --license ${LICENSE} \
  -m ${MAINTAINER} --description ${DESCRIPTION} --url ${URL} --vendor ${VENDOR} \
  --after-install etc/after-install.sh --before-remove etc/before-remove.sh \
  --after-remove etc/after-remove.sh --config-files etc/ ${SOURCES}

all: build

build:
	@$(MAKE) ${BTARGET} $(MFLAGS) -C nssplugin
	@go get ${GOFLAGS} ./...

debug: BTARGET:=debug
debug: build

test:
	@$(MAKE) ${TTARGET} $(MFLAGS) -C nssplugin
	@go test ${TFLAGS} ./...

cover: TFLAGS:=-cover
cover: TTARGET:=cover
cover: test

package: build
	@mkdir -p pkg
	@fpm -t deb -p pkg/ \
	  -d "libc6 >= ${LIBC_VERSION}" -d "libstdc++6 >= ${LIBSTDCXX_VERSION}" \
	  ${FPM_ARGS}
	@fpm -t rpm -p pkg/ \
	  -d "glibc >= ${LIBC_VERSION}" -d "libstdc++ >= ${LIBSTDCXX_VERSION}" \
	  ${FPM_ARGS}

clean:
	@rm -rf pkg
	@go clean -i ./...
	@$(MAKE) $@ $(MFLAGS) -C nssplugin
