#!/bin/bash
set -e
script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
"${script_dir}"/test-generate.sh
"${script_dir}"/../devpacker build "test_image" \
    -v \
    -p "${script_dir}/test-project" \
    --pull-policy if-not-present \
    --builder ghcr.io/chuxel/devcontainer-features/builder-devcontainer-empty \
    --trust-builder
