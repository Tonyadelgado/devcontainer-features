#!/bin/bash
set -e
script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
devpacker_dir="${script_dir}/../../devpacker"
devpack_root="${devpacker_dir}"/test/out/buildpack
prodpack_root="${script_dir}/.."

"${devpacker_dir}"/scripts/compile.sh true
"${devpacker_dir}"/devpacker generate "${script_dir}"/../.. "${devpack_root}"

"${script_dir}"/../../builders/create-builders.sh empty
cd "${script_dir}/test-project"
pack build -v prod_test_image \
    --pull-policy if-not-present \
    --builder ghcr.io/chuxel/devcontainer-features/builder-prod-empty \
    --trust-builder \
    --buildpack "${devpack_root}" \
    --buildpack "${prodpack_root}/npm"
