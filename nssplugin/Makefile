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
.PHONY: all build debug test cover mkdir clean rmobj

CC ?= g++
CFLAGS ?= -Wall -Wextra -O2 -fPIC -D_FORTIFY_SOURCE=2 -fstack-protector-all -Wa,--noexecstack -Wformat -Wformat-security -DSOCKET_PATH="\"$(SOCKET_PATH)\""
DIRS := obj bin gtest
GTEST := /usr/src/gtest
SOCKET_PATH := /var/run/gcua.socket

all: build

build: rmobj mkdir bin/libnss_google.so.2.0.1

debug: CFLAGS:=$(CFLAGS:-O2=-ggdb)
debug: build

test: SOCKET_PATH:=/tmp/compute_accounts_utils_test
test: CFLAGS:=$(CFLAGS:-O2=-ggdb) -fprofile-arcs -ftest-coverage
test: rmobj mkdir bin/utils_test
	@bin/utils_test --gtest_color=yes

cover: test
	@gcov utils.cc -o obj | grep \'utils.cc\' -A 1

mkdir:
	@mkdir -p $(DIRS)

clean:
	@rm -rf $(DIRS) *.gcov

rmobj:
	@rm -rf obj


# Link
bin/libnss_google.so.2.0.1: obj/libnss_google.o obj/utils.o
	$(CC) -o $@ -shared -Wl,-soname,libnss_google.so.2,-z,relro,-z,now $^

bin/utils_test: gtest/gtest-all.o gtest/gtest_main.o obj/utils_test.o obj/utils.o
	$(CC) -o $@ $^ -lpthread -lgcov

# Compile
obj/%.o: %.cc
	$(CC) -o $@ -c $(CFLAGS) $^

# GTest Compile
gtest/gtest-all.o: $(GTEST)/src/gtest-all.cc
	$(CC) -o $@ -c -I$(GTEST) $^

gtest/gtest_main.o: $(GTEST)/src/gtest_main.cc
	$(CC) -o $@ -c -I$(GTEST) $^
