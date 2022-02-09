#!/bin/bash
set -e
# The install.sh script is the installation entrypoint for any features in this repository. 
DEV_CONTAINER_FEATURE_SMOKE_TEST="${1:-"${DEV_CONTAINER_FEATURE_SMOKE_TEST-false}"}"
DEV_CONTAINER_CONFIG_DIR="/usr/local/etc/vscode-dev-containers"
DEV_CONTAINER_PROFILE_D="${DEV_CONTAINER_CONFIG_DIR}/profile.d"

if [ "$(id -u)" -ne 0 ]; then
    echo -e 'Script must be run as root. Use sudo, su, or add "USER root" to your Dockerfile before running this script.'
    exit 1
fi

# Move to the same directory as this script
cd "$(dirname "${BASH_SOURCE[0]}")"

# Import common utils
. ./common/utils.sh

# The tooling will parse the features.json + user devcontainer, and write 
# any build-time arguments into a feature-set scoped "devcontainer-features.env"
# The author is free to source that file and use it however they would like.
set -a
. ./devcontainer-features.env
set +a

# Syntax: conditional_install <feature_id>
# Executes feature's scripts if _BUILD_ARG_<FEATURE_ID> is set. It will
# automatically change the feature_id to upper case and swap out - for _. It
# expects that there is a folder named <feature_id> that contains the scripts.
conditional_install() {
    local feature_id="$1"
    get_buld_arg_env_var_name "${feature_id}"
    if [ -z "${!__retval}" ]; then
        return 0
    fi
    local feature_bin_dir="./features/${feature_id}/bin"

    # Always set profile.d folder name - buildpacks will also set this to their own values
    local feature_id_safe="$(echo "${feature_id}" | tr '[:lower:]' '[:upper:]' | tr '-' '_' )"
    profile_d_build_arg_name="_BUILD_ARG_${feature_id_safe}_PROFILE_D"
    declare -x ${profile_d_build_arg_name}="${DEV_CONTAINER_PROFILE_D}"

    echo "(*) Enabling feature \"$1\"..."
    chmod +x "${feature_bin_dir}"/*
    run_if_exists "${feature_bin_dir}/validate-prereqs"
    run_if_exists "${feature_bin_dir}/acquire"
    run_if_exists "${feature_bin_dir}/configure"
    if [ "${DEV_CONTAINER_FEATURE_SMOKE_TEST}" = "true" ] && [ -e "${feature_bin_dir}/test" ]; then
        echo "(*) Testing feature \"$1\"..."
        ${feature_bin_dir}/test
        echo "Passed!"
    fi
    echo
}

# Inject profile.d processing script into /etc/profile.d, /etc/bash.bashrc, /etc/zsh/zshrc
# for scenarios where they are used in a feature that is not installed via the buildpack.
# This makes it compatible with the buildpack spec's support for the same idea. We could
# in concept just adopt this as the approach for dev container features in general as well.
mkdir -p "${DEV_CONTAINER_PROFILE_D}"
if [ ! -e "${DEV_CONTAINER_CONFIG_DIR}/profile.d.sh" ]; then
cat << EOF > "${DEV_CONTAINER_CONFIG_DIR}/profile.d.sh"
if [ -d "${DEV_CONTAINER_PROFILE_D}" ] && [ -z "\${DEV_CONTAINER_PROFILE_D_DONE}" ]; then
    for script in "${DEV_CONTAINER_PROFILE_D}"/*.sh; do
        if [ -r "\$script" ]; then
            . \$script
        fi
        unset script
    done
    export DEV_CONTAINER_PROFILE_D_DONE="true"
fi
EOF
fi
symlink_if_ne ${DEV_CONTAINER_CONFIG_DIR}/profile.d.sh /etc/profile.d/9999-vscdc-profile.sh
snippet=". ${DEV_CONTAINER_CONFIG_DIR}/profile.d.sh"
echo "${snippet}" >> /etc/bash.bashrc
if [ -e "/etc/zsh" ]; then
    echo "${snippet}" >> /etc/zsh/zshrc
    echo "${snippet}" >> /etc/zsh/zprofile
fi

conditional_install buildpack-test
conditional_install python
conditional_install nodejs
conditional_install packcli
conditional_install vscode
conditional_install googlechrome

echo "(*) Done!"
