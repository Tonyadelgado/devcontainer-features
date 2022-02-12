#!/bin/bash
set -e
script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
buildpackify_dir="${script_dir}/../../buildpack"
devpack_root="${buildpackify_dir}"/test/out/buildpack
prodpack_root="${script_dir}/.."

"${buildpackify_dir}"/scripts/compile.sh
"${buildpackify_dir}"/buildpackify "${script_dir}"/../.. "${devpack_root}"

"${script_dir}"/../../builders/create-builders.sh empty
cd "${script_dir}/test-project"
pack build -v prod_test_image \
    --pull-policy if-not-present \
    --builder ghcr.io/chuxel/devcontainer-features/builder-prod-empty \
    --trust-builder \
    --buildpack "${devpack_root}" \
    --buildpack "${prodpack_root}/npm"
