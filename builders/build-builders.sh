#!/bin/bash
set -e
builder_type="${1:-"empty"}"

cd "$(dirname "${BASH_SOURCE[0]}")"

./build-stack-images.sh
pack builder create ghcr.io/chuxel/devcontainer-features/builder-devcontainer -c ${builder_type}/builder-devcontainer.toml
pack builder create ghcr.io/chuxel/devcontainer-features/builder-prod -c ${builder_type}/builder-prod.toml
