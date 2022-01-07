#!/bin/bash
set -e
PACK_CLI_VERSION="${1:-latest}"

# Import common utils
. "$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/scripts/utils.sh"

check_packages curl ca-certificates
if ! type git > /dev/null 2>&1; then
    apt-get -y install --no-install-recommends git
fi

echo "Downloading the Pack CLI..."
repo_url="https://github.com/buildpacks/pack"
find_version_from_git_tags PACK_CLI_VERSION "${repo_url}"
filename="pack-v${PACK_CLI_VERSION}-linux.tgz"
dl_url="${repo_url}/releases/download/v${PACK_CLI_VERSION}/${filename}"
mkdir -p /tmp/pack-cli
curl -sSL "${dl_url}" > /tmp/pack-cli/${filename}
curl -sSL "${dl_url}.sha256" > /tmp/pack-cli/${filename}.sha256
cd /tmp/pack-cli
sha256sum --ignore-missing -c "${filename}.sha256"
tar -f "${filename}" -C /usr/local/bin/ --no-same-owner -xzv pack
rm -rf /tmp/pack-cli
