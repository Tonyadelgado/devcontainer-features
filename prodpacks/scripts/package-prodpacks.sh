#!/bin/bash
set -e
cd "$(dirname "${BASH_SOURCE[0]}")"/..
export DOCKER_BUILDKIT=1
publish="${1:-false}"

publisher="$(jq -r '.publisher' ../buildpack-settings.json)"
featureset_name="$(jq -r '.featureSet' ../buildpack-settings.json)"
version="$(jq -r '.version' ../buildpack-settings.json)"

for pack_name in *; do
    if [  -d "${pack_name}" ]; then
        uri="ghcr.io/${publisher}/${featureset_name}/prodpack/${pack_name}:${version}"
        echo "(*) Packaging ${pack_name} prodpack as ${uri}..."
        pack buildpack package "${uri}" --pull-policy if-not-present -p ${pack_name}
        if [ "${publish}" = "true" ]; then
            # Expects that you are already logged in appropriatley
            echo "(*) Publishing..."
            docker push "${uri}"
        fi
    fi
done

