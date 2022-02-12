#!/bin/bash
set -e
cd "$(dirname "${BASH_SOURCE[0]}")"
export DOCKER_BUILDKIT=1
builder_name="${1:-"empty"}"
publish="${2:-false}"

publisher="$(jq -r '.publisher' ../devpack-settings.json)"
featureset="$(jq -r '.featureSet' ../devpack-settings.json)"
version="$(jq -r '.version' ../devpack-settings.json)"

mkdir -p /tmp/builder-tmp

create_builder() {
    local builder_type=$1
    local toml_dir="$(pwd)/${builder_name}"
    local toml="$(cat "${toml_dir}/builder-${builder_type}.toml")"
    toml="${toml//\${publisher\}/${publisher}}"
    toml="${toml//\${featureset\}/${featureset}}"
    toml="${toml//\${version\}/${version}}"
    toml="${toml//\${toml_dir\}/${toml_dir}}"
    echo "${toml}" > /tmp/builder-tmp/builder-${builder_type}.toml
    local uri="ghcr.io/${publisher}/${featureset}/builder-${builder_type}-${builder_name}"
    pack builder create "${uri}" --pull-policy if-not-present -c /tmp/builder-tmp/builder-${builder_type}.toml
    if [ "${publish}" = "true" ]; then
        echo "(*) Publishing..."
        docker push "${uri}"
    fi
}

create_builder devcontainer
create_builder prod

rm -rf /tmp/builder-tmp
