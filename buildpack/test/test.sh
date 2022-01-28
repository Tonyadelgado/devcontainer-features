#!/bin/bash
set -e
cd "$(dirname "${BASH_SOURCE[0]}")"/..
./scripts/prep.sh

buildpack_root="$(cd "out" && pwd)"
pack build -v test_image \
    -e "BP_DEV_CONTAINER_FEATURE_PACKCLI=true" \
    -e "BP_DEV_CONTAINER_FEATURE_GOOGLECHROME=true" \
    --pull-policy if-not-present \
    --builder ghcr.io/chuxel/devcontainer-features/builder-devcontainer \
    --trust-builder \
    --buildpack "${buildpack_root}"
