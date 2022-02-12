#!/bin/bash
set -e

root_path="$(cd "$(dirname "${BASH_SOURCE[0]}")"/.. && pwd)"
only_current_arch="${1:-false}"
if [ "$2" != "" ]; then
    shift
    os_to_build="$@"
else
    os_to_build="linux darwin windows"
fi
os_to_build=("${os_to_build}")

if [ "${only_current_arch}" = "true" ]; then
    arch="$(uname -m)"
    case $arch in
        x86_64)
            arch=amd64
            ;;
        armv8l | aarch64)
            arch=arm64
            ;;
        *)
            echo "Unsupported architecture: $arch"
            exit 1
            ;;
    esac
else 
    arch="amd64 arm64"
fi
arch=("${arch}")

build_api_binary()
{
    source="$1"
    binary_name="${2:-"${source}"}"
    cd "${root_path}/${source}"
    go get

    for target_os in ${os_to_build[@]}; do
        for target_arch in ${arch[@]}; do
            echo "Compiling ${target_os} ${target_arch}..."
            extn=""
            if [ "${target_os}" = "windows" ]; then
                extn=".exe"
            fi
            GOARCH="${target_arch}" GOOS="${target_os}" go build -a -o "${root_path}/dist/${binary_name}-${target_os}-${target_arch}${extn}"
        done
    done
}

mkdir -p "${root_path}"/dist/
build_api_binary src devpacker
chmod +x "${root_path}"/dist/*

