#!/bin/bash
set -e
script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

osname="$(uname)"
case $osname in
    Darwin | darwin)
        osname=darwin
        ;;
    Linux | linux | GNU/Linux)
        osname=linux
        ;;
    *)
        echo "Unsupported OS: $osname"
        exit 1
        ;;
esac
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

binary_name="devpacker-${osname}-${arch}"

"${script_dir}"/test-generate.sh
"${script_dir}"/../devpacker finalize --mode "devcontainer" finalize "test_image" "${script_dir}/test-project"
