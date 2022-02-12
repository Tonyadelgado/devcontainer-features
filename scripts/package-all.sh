#!/bin/bash
set -e
cd "$(dirname "${BASH_SOURCE[0]}")"
publish="${1:-false}"

./package-devpack.sh "${publish}"
./package-buildpacks.sh prodpacks "${publish}"
./package-buildpacks.sh modepacks "${publish}"
../devpacker/scripts/create-archives.sh
