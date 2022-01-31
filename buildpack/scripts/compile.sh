#!/bin/bash
set -e
root_path="$(cd "$(dirname "${BASH_SOURCE[0]}")"/.. && pwd)"

build_api_binary()
{
    source="$1"
    binary_name="${2:-"${source}"}"
    cd "${root_path}/${source}"
    GOARCH=amd64 GOOS=linux go build -a -o "${root_path}/dist/${binary_name}-amd64"
    GOARCH=arm64 GOOS=linux go build -a -o "${root_path}/dist/${binary_name}-arm64"
}

mkdir -p "${root_path}"/dist/
echo "Compiling go modules..."
build_api_binary src buildpackify
chmod +x "${root_path}"/dist/*

