#!/bin/bash
set -e
cd "$(dirname "${BASH_SOURCE[0]}")"

../scripts/create-full-builders.sh
cd assets
pack build test_image --pull-policy if-not-present --trust-builder  --builder ghcr.io/chuxel/devcontainer-features/builder-devcontainer
