#!/bin/bash
set -e
# The install.sh script is the installation entrypoint for any features in this repository. 
DEV_CONTAINER_FEATURE_SMOKE_TEST="${1:-"${DEV_CONTAINER_FEATURE_SMOKE_TEST-false}"}"

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
    echo "(*) Enabling feature \"$1\"..."
    chmod +x "${feature_bin_dir}"/*
    run_if_exists "${feature_bin_dir}/validate-prereqs"
    run_if_exists "${feature_bin_dir}/acquire"
    run_if_exists "${feature_bin_dir}/configure"
    if [ "${DEV_CONTAINER_FEATURE_SMOKE_TEST}" = "true" ] && [ -e "${feature_bin_dir}/test" ]; then
        echo
        echo "(*) Testing feature \"$1\"..."
        run_if_exists "${feature_bin_dir}/test"
        echo "Passed!"
    fi
    echo
}

conditional_install vscode
conditional_install googlechrome
conditional_install packcli

echo "(*) Done!"
