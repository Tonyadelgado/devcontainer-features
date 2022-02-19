#!/bin/bash
set -e
script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
devpacker_dir="${script_dir}/../devpacker"

clear_cache_flag=""
if [ "${1:-false}" = "true" ]; then
    clear_cache_flag="--clear-cache"
fi

test_project_folder="${2:-"${script_dir}/test-project"}"

"${script_dir}/../scripts/create-full-builders.sh"
"${devpacker_dir}/devpacker" build test_image \
    -p "${test_project_folder}" \
    ${clear_cache_flag} \
    --pull-policy if-not-present \
    --builder ghcr.io/chuxel/devcontainer-features/builder-devcontainer-full \
    --trust-builder
