#!/bin/bash
set -e
export DOCKER_BUILDKIT=1
cd "$(dirname "${BASH_SOURCE[0]}")"
publish="${1:-false}"

publisher="$(jq -r '.publisher' ../buildpack-settings.json)"
featureset_name="$(jq -r '.featureSet' ../buildpack-settings.json)"
version="$(jq -r '.version' ../buildpack-settings.json)"
uri_prefix="ghcr.io/${publisher}/${featureset_name}"

# Create two stacks - normal, devcontainer
echo "(*) Creating Stack images..."
docker build -t "${uri_prefix}/stack-build-image" --cache-from "${uri_prefix}/stack-build-image" --target build .
docker build -t "${uri_prefix}/stack-run-image" --cache-from "${uri_prefix}/stack-run-image" --target run .
docker build -t "${uri_prefix}/stack-devcontainer-build-image" --cache-from "${uri_prefix}/stack-devcontainer-build-image" --target devcontainer-build .
docker build -t "${uri_prefix}/stack-devcontainer-run-image" --cache-from "${uri_prefix}/stack-devcontainer-run-image" --target devcontainer-run .

if [ "${publish}" = "true" ]; then
    echo "(*) Publishing..."
    docker push "${uri_prefix}/stack-build-image"
    docker push "${uri_prefix}/stack-run-image"
    docker push "${uri_prefix}/stack-devcontainer-build-image"
    docker push "${uri_prefix}/stack-devcontainer-run-image"
fi
