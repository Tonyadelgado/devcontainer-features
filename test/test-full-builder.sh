#!/bin/bash
set -e
cd "$(dirname "${BASH_SOURCE[0]}")"

clear_cache_flag=""
if [ "${1:-false}" = "true" ]; then
    clear_cache_flag="--clear-cache"
fi

../scripts/create-full-builders.sh
cd assets
pack build test_image "${clear_cache_flag}" --pull-policy if-not-present --trust-builder  --builder ghcr.io/chuxel/devcontainer-features/builder-devcontainer
