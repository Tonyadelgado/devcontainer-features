#!/bin/bash
set -e
script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
buildpack_root="${script_dir}"/out/buildpack
mkdir -p "${buildpack_root}"

"${script_dir}"/../scripts/compile.sh
"${script_dir}"/../buildpackify "${script_dir}"/../.. "${buildpack_root}"

cd "${script_dir}/test-project"
pack build -v test_image \
    -e "BP_CONTAINER_FEATURE_PACKCLI=true" \
    -e "BP_CONTAINER_FEATURE_PACKCLI_VERSION=0.23.0" \
    -e "BP_CONTAINER_FEATURE_VSCODE=true" \
    -e "BP_CONTAINER_BUILD_MODE=devcontainer" \
    --pull-policy if-not-present \
    --builder ghcr.io/chuxel/devcontainer-features/builder-devcontainer \
    --trust-builder \
    --buildpack "${buildpack_root}"
