#!/bin/bash
set -e
# The install.sh script is the installation entrypoint for any features in this repository. 

FEATURE_REPOSITORY="github.com/chuxel/devcontainer-features"

if [ "$(id -u)" -ne 0 ]; then
    echo -e 'Script must be run as root. Use sudo, su, or add "USER root" to your Dockerfile before running this script.'
    exit 1
fi

# The tooling will parse the features.json + user devcontainer, and write 
# any build-time arguments into a feature-set scoped "devcontainer-features.env"
# The author is free to source that file and use it however they would like.
set -a
. ./devcontainer-features.env
set +a

# Source utilities
. "$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/scripts/utils.sh"

if [ ! -z ${_BUILD_ARG_VSCODE} ]; then
    run_script features/vscode.sh ${_BUILD_ARG_VSCODE_EDITION:-stable}
fi

if [ ! -z ${_BUILD_ARG_GOOGLECHROME} ]; then
    run_script features/chrome.sh
fi

if [ ! -z ${_BUILD_ARG_PACKCLI} ]; then
    run_script features/pack-cli.sh ${_BUILD_ARG_PACKCLI_VERSION:-latest}
fi

echo "Done!"
