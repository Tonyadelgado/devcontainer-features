#!/bin/bash
set -e
cd "$(dirname "${BASH_SOURCE[0]}")"/..

./builders/build-stack.sh
./builders/create-bulders.sh
./buildpack/test.sh
