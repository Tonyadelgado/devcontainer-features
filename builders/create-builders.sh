#!/bin/bash
set -e
builder_type="${1:-"empty"}"

cd "$(dirname "${BASH_SOURCE[0]}")"

pack builder create ghcr.io/chuxel/devcontainer-features/builder-devcontainer --pull-policy if-not-present -c ${builder_type}/builder-devcontainer.toml
pack builder create ghcr.io/chuxel/devcontainer-features/builder-prod --pull-policy if-not-present -c ${builder_type}/builder-prod.toml
