#!/bin/bash
set -e
cd "$(dirname "${BASH_SOURCE[0]}")"/..

build_api_binary()
{
    binary_name="$1"
    cd api-binaries/${binary_name}
    go build -o ../../out/bin/${binary_name}
    cd ../../
    chmod +x "out/bin/${binary_name}"
}

mkdir -p out/bin
build_api_binary build
build_api_binary detect

ln -sf "$(realpath ../features)" out/features
ln -sf "$(realpath ../common)" out/common
cp -f ../features.json out/
cp -f ../buildpack-settings.json out/
