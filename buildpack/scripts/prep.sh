#!/bin/bash
set -e
cd "$(dirname "${BASH_SOURCE[0]}")"/..
target_folder="${1:-out}"

mkdir -p "${target_folder}"

cp -rf ../features "${target_folder}"/
cp -rf ../common "${target_folder}"/
cp -f ../features.json "${target_folder}"/
cp -f ../buildpack-settings.json "${target_folder}"/
cp -rf assets/* "${target_folder}"/
