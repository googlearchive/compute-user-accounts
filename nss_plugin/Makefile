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

CC=g++
CFLAGS=-Wall -Wextra -ggdb -fPIC -D_FORTIFY_SOURCE=2 -fstack-protector-all -Wa,--noexecstack -Wformat -Wformat-security -DSOCKET_PATH="\"$(SOCKET_PATH)\""

DIRS=obj/ objt/ bin/
BIN=libnss_google.so.2.0.1
TESTS=utils_test
GTEST=/usr/src/gtest
SOCKET_PATH=/var/run/compute_accounts/sock

all: clean
all: mkdir
all: $(BIN:%=bin/%)

release: CFLAGS:=$(CFLAGS:-ggdb=-O2)
release: all

tests: SOCKET_PATH:=/tmp/compute_accounts_utils_test
tests: CFLAGS:=$(CFLAGS) -fprofile-arcs -ftest-coverage
tests: clean
tests: mkdir
tests: $(TESTS:%=bin/%)

mkdir:
	mkdir -p $(DIRS)

clean:
	rm -rf $(DIRS) *.gcov

# Link
bin/libnss_google.so.2.0.1: obj/libnss_google.o obj/utils.o
	$(CC) -o $@ -shared -Wl,-soname,libnss_google.so.2,-z,relro,-z,now $^

# Compile
obj/%.o: %.cc
	$(CC) -o $@ -c $(CFLAGS) $^

# Test Link
bin/utils_test: objt/gtest-all.o objt/gtest_main.o objt/utils_test.o obj/utils.o
	$(CC) -o $@ $^ -lpthread -lgcov

# Test Compile
objt/gtest-all.o: $(GTEST)/src/gtest-all.cc
	$(CC) -o $@ -c $(CFLAGS) -I$(GTEST) $^

objt/gtest_main.o: $(GTEST)/src/gtest_main.cc
	$(CC) -o $@ -c $(CFLAGS) -I$(GTEST) $^

objt/%.o: %.cc
	$(CC) -o $@ -Isrc -c $(CFLAGS) $^

.PHONY: all release tests mkdir clean
