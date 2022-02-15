#!/bin/bash
set -e
script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

cd "${script_dir}/../.."
./scripts/create-full-builders.sh
./devpacker/devpacker build prod_test_image \
    -v \
    --path "${script_dir}/test-project" \
    --pull-policy if-not-present \
    --builder ghcr.io/chuxel/devcontainer-features/builder-prod-full \
    --trust-builder
