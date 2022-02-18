#!/bin/bash
set -e
script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
buildpack_root="${script_dir}"/out/buildpack
mkdir -p "${buildpack_root}"

clear_cache_flag=""
if [ "${1:-false}" = "true" ]; then
    clear_cache_flag="--clear-cache"
fi

"${script_dir}"/../scripts/compile.sh true
"${script_dir}"/../devpacker generate "${script_dir}"/../.. "${buildpack_root}"

pack build test_image \
    -v \
    -e "BP_CONTAINER_FEATURE_BUILDPACK_TEST=true" \
    -e "BP_CONTAINER_FEATURE_BUILDPACK_TEST_FOO=bar-override" \
    -p "${script_dir}/test-project" \
    --pull-policy if-not-present \
    --builder ghcr.io/chuxel/devcontainer-features/builder-devcontainer-empty \
    --trust-builder \
    ${clear_cache_flag} \
    --buildpack "${script_dir}/../../modepacks/devcontainer" \
    --buildpack "${buildpack_root}" \
