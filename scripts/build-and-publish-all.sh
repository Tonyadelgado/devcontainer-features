#!/bin/bash
set -e
cd "$(dirname "${BASH_SOURCE[0]}")"
token_env_var_name="${1:-GITHUB_TOKEN}"
username="${2:-chuxel}"
registry="${3:-ghcr.io}"

echo "${!token_env_var_name}" | docker login "${registry}" -u "${username}" --password-stdin
./create-full-builders.sh true
