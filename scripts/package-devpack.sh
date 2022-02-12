#!/bin/bash
set -e
cd "$(dirname "${BASH_SOURCE[0]}")"/..
export DOCKER_BUILDKIT=1
publish="${1:-false}"

# Create and package devpack
publisher="$(jq -r '.publisher' devpack-settings.json)"
featureset_name="$(jq -r '.featureSet' devpack-settings.json)"
version="$(jq -r '.version' devpack-settings.json)"
uri="ghcr.io/${publisher}/${featureset_name}/devpack:${version}"

./devpacker/scripts/compile.sh false

echo "(*) Generating devpack from dev container features..."
mkdir -p /tmp/buildpack-out
./devpacker/devpacker generate "." /tmp/buildpack-out

echo "(*) Packaging devpack as ${uri}..."
cd /tmp/buildpack-out
echo -e '[buildpack]\nuri = "."' > /tmp/buildpack-out/package.toml
pack buildpack package "${uri}" --pull-policy if-not-present -p /tmp/buildpack-out
cd ..
rm -rf /tmp/buildpack-out

# Expects that you are already logged in appropriatley
if [ "${publish}" = "true" ]; then
    echo "(*) Publishing..."
    docker push "${uri}"
fi
