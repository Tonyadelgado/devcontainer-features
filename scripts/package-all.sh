#!/bin/bash
set -e
script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
devpacker_dir="${script_dir}/../devpacker"

publish="${1:-false}"

"${script_dir}"/package-devpack.sh "${publish}"
"${script_dir}"/package-buildpacks.sh prodpacks "${publish}"
"${script_dir}"/package-buildpacks.sh modepacks "${publish}"
"${script_dir}"/package-builders.sh "empty" "${publish}"
"${script_dir}"/package-builders.sh "full" "${publish}"
"${devpacker_dir}"/scripts/create-archives.sh
