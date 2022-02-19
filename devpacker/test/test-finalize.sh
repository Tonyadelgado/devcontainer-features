#!/bin/bash
set -e
script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
"${script_dir}"/test-generate.sh "${1:-false}"
"${script_dir}"/../devpacker finalize "test_image" "${2:-"${script_dir}/test-project"}"
