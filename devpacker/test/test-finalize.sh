#!/bin/bash
set -e
script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
"${script_dir}"/test-generate.sh
"${script_dir}"/../devpacker finalize "test_image" "${script_dir}/test-project"
