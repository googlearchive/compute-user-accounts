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
