# Chuxel's Dev Container Feature Sandbox

*Everything here will change - a lot. Don't depend on any of it for anything you're doing. Good for demos, that's it.*

## Development setup

1. Open this repository in GitHub Codespaces or Remote - Containers using VS Code
2. Once connected, open `devcontainer-features.code-workspace`

## Adding another feature

1. Update `/devcontainer-features.json` to add any feature configuration like extensions, settings, etc.
    1. Add a `targetPath` option with a default for when used outside of a Devpack. Typically this is `/usr/local`.
    1. Add a `buildMode` option if the feature needs to behave differently in production vs devcontainer mode.
2. Create a sub-folder under `/features` with a `bin` folder that contains one or more of the following scripts/binaries:
    - `acquire` - [Feature/Devpack] Main step for tools acquisition and installation. **May run as a user other than root, needs to take a path as input.** Specifcally, the path in the `_BUILD_ARG_<FEATURE_ID>_TARGETPATH` env var.
    - `configure` - [Feature/Devpack] Post-installation step for things that require root access to perform. Not executed by pack CLI when used as a Devpack, but instead using the dev container CLI or VS Code. Also create or symlink anything (from `_BUILD_ARG_<FEATURE_ID>_TARGETPATH`) that needs to have a consistent location here.
    - `detect` - [Devpack] Optional script to do detection if no other buildpacks or devcontainer.json (in devcontainer build mode) reference it.
    - `verify-prereqs` - [Feature] Runs before acquire to install any dependencies needed acquire the tool. Skipped in a Devpack since these prerequisites need to be in the build "stack" image.
2. Ensure all scripts/binaries have the execute bit set (chmod +x).

### Environment variables for `acquire` and `configure`

The `acquire` and `configure` scripts will have some environment passed into them it:

- `_BUILD_ARG_<FEATUREID>_BUILDMODE` - Allows the script to alter behavior depending on what the feature is being used for. Will be either `devcontainer` when the build is to create a dev container image or `production` for non-development scenarios.
- `_BUILD_ARG_<FEATUREID>_TARGETPATH` - Location to install the tool. Include symlinks to `bin` in this folder to ensure they are in the path if you do not directly install there.
- `_BUILD_ARG_<FEATUREID>_PROFILE_D` - Location you can place any executable (chmod +x) scripts that should be sourced from an interactive or login shell. Note that this is just used to resolve environment variables, not bring new functions into the shell.
- `_BUILD_ARG_<FEATUREID>_<OPTION>` - Any selections made based on options in `devcontainer-features.json`. Mirrors what would be in `devcontainer-features.env` in `install.sh`. When used in a Devpack, you can also set these variables in `project.toml` or the `pack` CLI using `BP_CONTAINER_FEATURE_<FEATUREID>_<OPTION>` and will always be applied and setting `BP_CONTAINER_FEATURE_<FEATUREID>` to `true` will enable the feature regardless. Values in `devcontainer.json` are only considered when `_BUILD_ARG_<FEATUREID>_BUILD_MODE` is set to `devcontainer`.

### Environment variables for `detect`

The detect script is only used in Devpacks and therefore has a few different environment variables passed into it.

- `_BUILD_ARG_<FEATUREID>_BUILDMODE` - Allows the script to alter behavior depending on what the feature is being used for. Will be either `devcontainer` when the build is to create a dev container image or `production` for non-development scenarios.
- `_BUILD_ARG_<FEATUREID>_<OPTION>` - Any selections already made based on options in `devcontainer-features.json`. Mirrors what would be in `devcontainer-features.env` in `install.sh`.
-`_BUILD_ARG_<FEATUREID>_SELECTION_ENV_FILE_PATH` - Specifies the location of a `.env` file that can be used to update feature options that should be passed as build arguments to the `acquire` or `configure` scripts. Add these selections in the file using the same `_BUILD_ARG_<FEATUREID>_<OPTION>` variables the other scripts expect as inputs. However, note that `...TARGETPATH`, `...PROFILE_D`, and `...BUILDMODE` variables cannot be updated.

Any features set up this way will be automatically included in the next repository release.

## Adding another Buildpack (prodpack)

You can add another Buildpack to this repository by creating a folder under `/prodpacks`. Each folder should follow the standard [Buildpack spec](https://buildpacks.io/).

### Adding a dependency on a feature

Beyond the standard Buildpack spec, note that you can reference any features in this repository along with setting options or them as a dependency in your `detect` script. For example, consider the following:

```bash
#!/usr/bin/env bash
set -euo pipefail
platform_dir=$1
build_plan=$2

node_version="16.14.0"

cat >> "${build_plan}" << EOF
[[requires]]
  name = "chuxel/devcontainer-features/nodejs"
  [requires.metadata]
    build = true
    launch = true
    option_version="${node_version}"
EOF
exit 0
```

This will update the build plan to enable the feature stored in `/features/nodejs` and set the the `version` option to the value of `node_version`. This will result in `_BUILD_ARG_NODEJS_VERSION` being set in this feature's `acquire` and `configure` scripts. You can use the same pattern for other options.

Setting `build=true` and `launch=true` ensures the output is included in the image used during image build the resulting output "launch" image. You can also control caching but setting `launch` to `true` or `false`

### Adding the Buildpack to the full builder

To add a Buildpack under the `/prodpack` folder to the full production builder, update `/builders/full/builder-prod.toml` as follows:

1. Add a `[[buildpacks]]` entry to the top of the file pointing to the folder with your Buildpack in it. For example, this references the npm Buildpack: 

    ```toml
    [[buildpacks]]
      id = "${publisher}/${featureset}/prodpacks/npm"
      uri = "${toml_dir}/../../prodpacks/npm"
    ```

1. Add id to an `[[order.group]]` entry under `[[order]]`. For example: 

    ```toml
    [[order.group]]
    id = "${publisher}/${featureset}/prodpacks/npm"
    ```

## Releasing

To release an update to the features, Devpack, Buildpacks, stack images, devpacker CLI, and Builders, follow these steps:

1. Increase the version number in `devpack-settings.json`
2. Commit and push
3. Run `git tag <version-you-added-to-devpack-settings.json>`
4. Run `git push --tags`

GitHub Actions will take care of the rest.

## Using the devpacker CLI

There are scripts in this repository under the `scripts` and `test` folders under the root (`/`), `/devpacker` and `prodpacks` folders to do most things you'll want to do.

To use the `devpacker` CLI locally, go to the repository releases and download the .zip / .tgz file for your system. Add / symlink `devpacker` or `devpacker.cmd` into your `PATH`.

You can use the full builders with the `devpacker` CLI locally as well. The `devpacker build` command accepts the same arguments as the `pack build`. For example, to use the production builder to produce an image called `prod_test_image`, execute the following from your project repo:

```bash
devpacker build prod_test_image --trust-builder --pull-policy if-not-present --builder ghcr.io/chuxel/devcontainer-features/builder-prod-full
```

Generally you could use the pack CLI here instead, but any post-processing finalization steps will not happen if needed.

For the dev container builder, just change the builder image. To do this with an image called `test_image`:

```bash
devpacker build test_image --trust-builder  --pull-policy if-not-present --builder ghcr.io/chuxel/devcontainer-features/builder-devcontainer-full
```

This will tweak the image and output a modified `devcontainer.json.devpack` file. You can rename this to `devcontainer.json` and open it up in Remote - Containers to finish post-processing.
