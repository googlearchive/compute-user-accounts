#!/bin/bash

set -e
make
make release
make tests
for test in bin/*; do
  echo $test
  $test
done
make clean
