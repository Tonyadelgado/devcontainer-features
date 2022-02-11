#!/bin/bash
set -e
cd "$(dirname "${BASH_SOURCE[0]}")"
export DOCKER_BUILDKIT=1
builder_type="${1:-"empty"}"
publish="${2:-false}"

publisher="$(jq -r '.publisher' ../buildpack-settings.json)"
featureset_name="$(jq -r '.featureSet' ../buildpack-settings.json)"
version="$(jq -r '.version' ../buildpack-settings.json)"
uri_prefix="ghcr.io/${publisher}/${featureset_name}"

echo "(*) Creating ${builder_type} Builder.."
pack builder create "${uri_prefix}/builder-devcontainer-${builder_type}" --pull-policy if-not-present -c ${builder_type}/builder-devcontainer.toml
pack builder create "${uri_prefix}/builder-prod-${builder_type}" --pull-policy if-not-present -c ${builder_type}/builder-prod.toml

if [ "${publish}" = "true" ]; then
    echo "(*) Publishing..."
    docker push "${uri_prefix}/builder-devcontainer-${builder_type}"
    docker push "${uri_prefix}/builder-prod-${builder_type}"
fi