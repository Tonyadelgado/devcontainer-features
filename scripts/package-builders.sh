#!/bin/bash
set -e
script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
builder_name="${1:-"full"}"
publish="${2:-false}"

"${script_dir}"/../builders/build-stack-images.sh "${publish}"
"${script_dir}"/../builders/create-builders.sh "${builder_name}" "${publish}"
