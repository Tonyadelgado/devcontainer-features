#!/bin/bash
set -e
buildpack_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/../out" && pwd)"
docker build -t devcontainer-run-stack-image --target devcontainer .
pack build -v test_image \
    --builder paketobuildpacks/builder:full \
    --run-image devcontainer-run-stack-image \
    --buildpack "${buildpack_root}"
