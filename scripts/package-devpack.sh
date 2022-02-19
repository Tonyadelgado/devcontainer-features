#!/bin/bash
set -e
export DOCKER_BUILDKIT=1
publish="${1:-false}"
script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
devcontainer_features_dir="${script_dir}/../devcontainer-features"
devpacker_dir="${script_dir}/../devpacker"

publisher="$(jq -r '.publisher' "${devcontainer_features_dir}"/devpack-settings.json)"
featureset_name="$(jq -r '.featureSet' "${devcontainer_features_dir}"/devpack-settings.json)"
version="$(jq -r '.version' "${devcontainer_features_dir}"/devpack-settings.json)"
uri="ghcr.io/${publisher}/${featureset_name}/devpack:${version}"

"${devpacker_dir}"/scripts/compile.sh false

echo "(*) Generating devpack from dev container features..."
mkdir -p /tmp/buildpack-out
"${devpacker_dir}"/devpacker generate "${devcontainer_features_dir}" /tmp/buildpack-out

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
