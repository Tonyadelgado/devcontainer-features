#!/bin/bash
set -e
script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
buildpack_root="${script_dir}"/out/buildpack
features_root="${script_dir}"/../..
mkdir -p "${buildpack_root}"/bin

# Copy binaries and scripts
cp -rf "${script_dir}"/../src/assets/* "${buildpack_root}"/
cp -f "${script_dir}"/../dist/* "${buildpack_root}"/bin/

# Copy test features
cp -rf "${features_root}"/features "${buildpack_root}"/
cp -rf "${features_root}"/common "${buildpack_root}"/
cp -f "${features_root}"/devcontainer-features.json "${buildpack_root}"/
cp -f "${features_root}"/buildpack-settings.json "${buildpack_root}"/

# Copy test buildpack.toml
cp -rf ${script_dir}/assets/buildpack.toml "${buildpack_root}"/
