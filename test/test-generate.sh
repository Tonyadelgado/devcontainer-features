#!/bin/bash
set -e
cd "$(dirname "${BASH_SOURCE[0]}")"/..

./builders/build-stack-images.sh
./builders/create-builders.sh
./devpacker/test/test-generate.sh
