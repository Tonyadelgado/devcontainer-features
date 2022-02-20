#!/bin/bash
set -e
export DOCKER_BUILDKIT=1
buildpack_type="${1:-prodpacks}"
publish="${2:-false}"
script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
devcontainer_features_dir="${script_dir}/../devcontainer-features"
packs_dir="${script_dir}/../${buildpack_type}"

publisher="$(jq -r '.publisher' "${devcontainer_features_dir}"/devpack-settings.json)"
featureset_name="$(jq -r '.featureSet' "${devcontainer_features_dir}"/devpack-settings.json)"
version="$(jq -r '.version' "${devcontainer_features_dir}"/devpack-settings.json)"

for pack_path in "${packs_dir}"/*; do
    if [  -d "${pack_path}" ]; then
        pack_name="$(basename "${pack_path}")"
        uri="ghcr.io/${publisher}/${featureset_name}/${buildpack_type}/${pack_name}:${version}"
        echo "(*) Packaging ${pack_name} as ${uri}..."
        pack buildpack package "${uri}" --pull-policy if-not-present -p ${pack_path}
        if [ "${publish}" = "true" ]; then
            # Expects that you are already logged in appropriatley
            echo "(*) Publishing..."
            docker push "${uri}"
        fi
    fi
done

