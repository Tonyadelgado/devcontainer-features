#!/bin/bash
set -e
cd "$(dirname "${BASH_SOURCE[0]}")"/..
rm -rf "out"
mkdir -p "out"

# Create windows zip
echo "(*) Creating devpacker-windows.zip"
zip -R "out/devpacker-windows.zip" "devpacker.cmd" "dist/devpacker-windows-*" "dist/devpacker-linux-*"

# Create macOS tgz
echo "(*) Creating devpacker-darwin.tgz"
tar -czvf "out/devpacker-darwin.tgz"  --exclude="./dist/devpacker-windows-*" "./devpacker" "./dist"

# Create Linux tgz
echo "(*) Creating devpacker-linux.tgz"
tar -czvf "out/devpacker-linux.tgz" --exclude="./dist/devpacker-windows-*" --exclude="./dist/devpacker-darwin-*"  "./devpacker" "./dist"
