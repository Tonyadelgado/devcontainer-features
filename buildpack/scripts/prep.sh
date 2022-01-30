#!/bin/bash
set -e
cd "$(dirname "${BASH_SOURCE[0]}")"/..

./scripts/compile.sh

cp -rf ../features out/
cp -rf ../common out/
cp -f ../features.json out/
cp -f ../buildpack-settings.json out/
cp -rf assets/* out/
