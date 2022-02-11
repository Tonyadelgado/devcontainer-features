#!/bin/bash
set -e
cd "$(dirname "${BASH_SOURCE[0]}")"/..
export DOCKER_BUILDKIT=1
publish="${1:-false}"

publisher="$(jq -r '.publisher' buildpack-settings.json)"
featureset_name="$(jq -r '.featureSet' buildpack-settings.json)"
version="$(jq -r '.version' buildpack-settings.json)"
uri="ghcr.io/${publisher}/${featureset_name}/buildpack:${version}"

./buildpack/scripts/compile.sh

echo "(*) Generating buildpack from dev container features..."
mkdir -p /tmp/buildpack-out
./buildpack/buildpackify "." /tmp/buildpack-out

echo "(*) Packaging buildpack as ${uri}..."
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