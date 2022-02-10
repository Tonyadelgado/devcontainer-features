#!/bin/bash
set -e
cd "$(dirname "${BASH_SOURCE[0]}")"/..

./scripts/package-buildpack.sh
./builders/build-stack-images.sh
./builders/create-builders.sh full
