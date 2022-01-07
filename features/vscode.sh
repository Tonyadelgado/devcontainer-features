#!/bin/bash
set -e
EDITION="${1:-stable}" # stable, insiders, both
MICROSOFT_GPG_KEYS_URI="https://packages.microsoft.com/keys/microsoft.asc"

# Import common utils
. "$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/scripts/utils.sh"

# Determine the appropriate non-root user
username="automatic"
detect_user username

if [ "$(id -u)" -ne 0 ]; then
    echo -e 'Script must be run as root. Use sudo, su, or add "USER root" to your Dockerfile before running this script.'
    exit 1
fi

install-code() {
    local url=$1
    local is_insiders=$2
    local cmd=/usr/bin/code
    local cfg='$HOME/.config/Code/User/'
    if [ "$is_insiders" = "true" ]; then
        cmd=/usr/bin/code-insiders
        cfg='$HOME/.config/Code - Insiders/User/'
    fi
    
    curl -sSL "$url" -o /tmp/code.deb
    apt-get install -y /tmp/code.deb
    rm -f /tmp/code.deb

    su ${username} -c "\
        echo \"Adding settings.json to $cfg...\" \
        && mkdir -p \"$cfg\" \
        && echo '{ \"window.titleBarStyle\": \"custom\" }' > \"$cfg/settings.json\"" 2>&1
}

export DEBIAN_FRONTEND=noninteractive
check_packages curl ca-certificates apt-transport-https dirmngr gnupg2

curl -sSL ${MICROSOFT_GPG_KEYS_URI} | gpg --dearmor > /usr/share/keyrings/microsoft-archive-keyring.gpg
echo "deb [arch=$(dpkg --print-architecture) signed-by=/usr/share/keyrings/microsoft-archive-keyring.gpg] https://packages.microsoft.com/repos/code stable main" > /etc/apt/sources.list.d/vscode.list
apt-get update

# Install VS Code
to_install=""
if [ "${EDITION}" != "insiders" ]; then
    to_install="code"
fi

if [ "${EDITION}" != "stable" ]; then
    to_install="${to_install} code-insiders"
fi
apt-get -y install ${to_install}
