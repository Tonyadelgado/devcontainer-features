#!/bin/bash
set -e
cd "$(dirname "${BASH_SOURCE[0]}")"/..

./scripts/compile.sh

ln -sf "$(realpath ../features)" out/features
ln -sf "$(realpath ../common)" out/common
cp -f ../features.json out/
cp -f ../buildpack-settings.json out/
cp -f assets/* out/
