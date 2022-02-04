#!/bin/bash
set -e
script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

"${script_dir}"/test-generate.sh
"${script_dir}"/../dist/buildpackify-amd64 --mode "devcontainer" finalize "test_image" "${script_dir}/test-project"
