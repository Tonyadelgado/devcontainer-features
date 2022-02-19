#!/bin/bash
set -e
test_image=${1:-"mcr.microsoft.com/vscode/devcontainers/base:ubuntu"}

cd "$(dirname "${BASH_SOURCE[0]}")/devcontainer-features"
docker run -it --rm -u root -v "$(pwd):/features" "${test_image}" bash /features/install.sh true