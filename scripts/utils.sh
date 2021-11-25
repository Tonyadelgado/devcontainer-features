# Function to run apt-get if needed
apt_get_update_if_needed()
{
    if [ ! -d "/var/lib/apt/lists" ] || [ "$(ls /var/lib/apt/lists/ | wc -l)" = "0" ]; then
        echo "Running apt-get update..."
        apt-get update
    else
        echo "Skipping apt-get update."
    fi
}

# Checks if packages are installed and installs them if not
check_packages() {
    if ! dpkg -s "$@" > /dev/null 2>&1; then
        apt_get_update_if_needed
        apt-get -y install --no-install-recommends "$@"
    fi
}

# If in automatic mode, determine if a user already exists, if not use vscode
detect_user() {
    local user_variable_name=${1:-username}
    local user_variable_value=${!user_variable_name}
    local possible_users=${2:-("vscode" "node" "codespace" "$(awk -v val=1000 -F ":" '$3==val{print $1}' /etc/passwd)")}
    local uid_variable_name=${3:-user_uid}
    local gid_variable_name=${4:-user_gid}
    if [ "${user_variable_value}" = "auto" ] || [ "${user_variable_value}" = "automatic" ]; then
        declare -g ${user_variable_name}=vscode
        for current_user in ${possible_users[@]}; do
            if id -u ${current_user} > /dev/null 2>&1; then
                declare -g ${user_variable_nam}e=${current_user}
                break
            fi
        done
        if [ "${user_variable_value}" = "" ]; then
            declare -g ${user_variable_name}=vscode
        fi
    elif [ "${user_variable_value}" = "none" ]; then
        declare -g ${user_variable_name}=root
        declare -g ${uid_variable_name}=0
        declare -g ${gid_variable_name}=0
    fi
}
