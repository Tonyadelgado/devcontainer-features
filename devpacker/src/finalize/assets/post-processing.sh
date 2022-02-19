#!/bin/bash
set -e
post_procesing_array=( "$@" )
echo "To post process: $@"

# The CNB_APP_DIR is filtered out by the launcher, so this is an easy way to detect if the current process
# was started via the launcher. In the event of an "docker exec", the containe entrypoint is not fired, so
# the shell process will not be launched properly. So, what this will do is detect if we're in a login or 
# interactive sh, bash, or zsh process and replace the shell with one started using /cnb/lifecycle/launcher
# with the exact same arguments. In devcontainer.json, we'll then force userEnvProbe to be loginInteractiveShell.
#
# More info on launcher environment: https://github.com/buildpacks/spec/blob/main/platform.md#launch-environment
snippet="$(cat << 'EOF'
# Ensure all interactive or login shells are initalized via /cnb/lifecycle/launcher (which is also the default entrypoint)
if [ ! -z "${CNB_APP_DIR}" ] && [ -z "${DCNB_ENV_LOADED}" ]; then export DCNB_ENV_LOADED=true; mapfile -d $'\0' _cmd_line < /proc/$$/cmdline; exec /cnb/lifecycle/launcher "${_cmd_line[@]//\"/\\\"}"; fi
EOF
)"

add_to_top_of_file() {
    local filename="$1"
    local check_exists="${2:-$1}"
    if [ ! -e "${check_exists}" ]; then
        echo "${check_exists} does not exist. Skipping."
        return
    fi
    local existing_file="$(cat "${filename}" 2>/dev/null)"
    if ! grep -Fxq "${snippet//\0/\\0}" "${filename}" 2>/dev/null; then
        echo "${snippet}
${existing_file}" > "${filename}"
        echo "Adding /cnb/lifecycle/launcher to ${filename}."
    else 
        echo "/cnb/lifecycle/launcher already exists in ${filename}. Skipping."
    fi
}

add_to_top_of_file /etc/bash.bashrc
add_to_top_of_file /etc/profile
add_to_top_of_file /etc/zsh/zshenv /etc/zsh

for feature in ${post_procesing_array[@]}; do
    feature_id="${feature##*\/}"
    buildpack_folder_name="${feature%\/*}"
    buildpack_folder_name="${buildpack_folder_name//\//_}"
    feature_layer_path="${CNB_LAYERS_DIR:-/layers}/${buildpack_folder_name}/${feature_id}"
    feature_config_path="${feature_layer_path}/etc/dev-container-features/feature-config/features/${feature_id}"

    echo "Processing: ${feature_config_path}"

    if [ -r "${feature_config_path}/bin/configure}" ]; then
        set -a
        . "${feature_config_path}/devcontainer-features.env"
        set +a
        chmod +x "${feature_config_path}/bin/configure"
        "${feature_config_path}/bin/configure"
    fi

    # Remove env vars for features since this will result in duplicates after post-processing
    rm -rf "${feature_layer_path}/env" "${feature_layer_path}/etc/dev-container-features/feature-config"
done

