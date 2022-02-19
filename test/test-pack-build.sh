#!/bin/bash
set -e
script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
devpacker_dir="${script_dir}/../devpacker"

"${script_dir}"/../builders/build-stack-images.sh
"${script_dir}"/../builders/create-builders.sh
"${devpacker_dir}"/test/test-pack-build.sh "${1:-false}" "${2:-"${script_dir}/test-project"}"

