#!/bin/bash
set -e
cd "$(dirname "${BASH_SOURCE[0]}")"/..
./scripts/prep.sh

buildpack_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/out" && pwd)"
pack build -v test_image \
    --pull-policy if-not-present \
    --builder ghcr.io/chuxel/devcontainer-features/builder-devcontainer \
    --trust-builder \
    --buildpack "${buildpack_root}"
