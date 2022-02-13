#!/bin/bash
set -e
export DOCKER_BUILDKIT=1
packs_dir="${1:-prodpacks}"
publish="${2:-false}"
cd "$(dirname "${BASH_SOURCE[0]}")/../${packs_dir}"

publisher="$(jq -r '.publisher' ../devpack-settings.json)"
featureset_name="$(jq -r '.featureSet' ../devpack-settings.json)"
version="$(jq -r '.version' ../devpack-settings.json)"

for pack_name in *; do
    if [  -d "${pack_name}" ] && [ "${pack_name}" != "test" ]; then
        uri="ghcr.io/${publisher}/${featureset_name}/${packs_dir}/${pack_name}:${version}"
        echo "(*) Packaging ${packs_dir}/${pack_name} as ${uri}..."
        pack buildpack package "${uri}" --pull-policy if-not-present -p ${pack_name}
        if [ "${publish}" = "true" ]; then
            # Expects that you are already logged in appropriatley
            echo "(*) Publishing..."
            docker push "${uri}"
        fi
    fi
done

