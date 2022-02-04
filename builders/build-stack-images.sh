#!/bin/bash
set -e
cd "$(dirname "${BASH_SOURCE[0]}")"

# Create two stacks - normal, devcontainer
echo "Creating Stack images..."
export DOCKER_BUILDKIT=1
docker build -t ghcr.io/chuxel/devcontainer-features/stack-build-image --target build .
docker build -t ghcr.io/chuxel/devcontainer-features/stack-run-image --target run .
docker build -t ghcr.io/chuxel/devcontainer-features/stack-devcontainer-build-image --target devcontainer-build .
docker build -t ghcr.io/chuxel/devcontainer-features/stack-devcontainer-run-image --target devcontainer-run .

