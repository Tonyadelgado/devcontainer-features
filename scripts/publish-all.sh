#!/bin/bash
set -e
script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
token_env_var_name="${1:-GITHUB_TOKEN}"
username="${2:-chuxel}"
registry="${3:-ghcr.io}"

echo "${!token_env_var_name}" | docker login "${registry}" -u "${username}" --password-stdin
"${script_dir}"/package-all.sh true
