#!/bin/bash
set -e
script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
buildpack_root="${script_dir}"/out/buildpack
mkdir -p "${buildpack_root}"

clear_cache_flag=""
if [ "${1:-false}" = "true" ]; then
    clear_cache_flag="--clear-cache"
fi

test_project_folder="${2:-"${script_dir}/test-project"}"

"${script_dir}"/../scripts/compile.sh true
"${script_dir}"/../devpacker generate "${script_dir}"/../../devcontainer-features "${buildpack_root}"
"${script_dir}"/../devpacker build "test_image" \
    -v \
    -p "${test_project_folder}" \
    --pull-policy if-not-present \
    --builder ghcr.io/chuxel/devcontainer-features/builder-devcontainer-empty \
    ${clear_cache_flag} \
    --trust-builder
