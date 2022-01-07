#!/bin/bash
set -e
cd "$(dirname "${BASH_SOURCE[0]}")"

sudo rm -rf /usr/local/etc/vscode-dev-containers/features/github.com/chuxel/devcontainer-features
sudo bash install.sh