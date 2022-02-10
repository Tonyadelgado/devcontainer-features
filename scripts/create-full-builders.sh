#!/bin/bash
set -e
cd "$(dirname "${BASH_SOURCE[0]}")"/..
publish="${1:-false}"

./scripts/package-buildpack.sh "${publish}"
./builders/build-stack-images.sh "${publish}"
./builders/create-builders.sh full "${publish}"
