#!/bin/bash
set -e
script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

cd "${script_dir}/../.."
./scripts/create-full-builders.sh
cd "${script_dir}/test-project"
pack build -v prod_test_image \
    --pull-policy if-not-present \
    --builder ghcr.io/chuxel/devcontainer-features/builder-prod-full \
    --trust-builder
