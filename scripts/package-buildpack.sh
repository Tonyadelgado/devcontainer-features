#!/bin/bash
set -e
cd "$(dirname "${BASH_SOURCE[0]}")"/..

buildpack_publisher="$(jq -r '.publisher' buildpack-settings.json)"
buildpack_featureset_name="$(jq -r '.featureSet' buildpack-settings.json)"
buildpack_version="$(jq -r '.version' buildpack-settings.json)"
buildpack_uri="ghcr.io/${buildpack_publisher}/${buildpack_featureset_name}/buildpack:${buildpack_version}"

./buildpack/scripts/compile.sh

echo "Generating buildpack from dev container features..."
mkdir -p /tmp/buildpack-out
./buildpack/buildpackify "." /tmp/buildpack-out

echo "Packaging buildpack as ${buildpack_uri}..."
cd /tmp/buildpack-out
echo -e '[buildpack]\nuri = "."' > /tmp/buildpack-out/package.toml
pack buildpack package "${buildpack_uri}" -c /tmp/buildpack-out/package.toml
cd ..
rm -rf /tmp/buildpack-out