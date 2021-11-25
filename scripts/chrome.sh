#!/bin/bash

# Import common utils
. "$(cd $(dirname "${BASH_SOURCE[0]}") && pwd)/utils.sh"

if [ "$(id -u)" -ne 0 ]; then
    echo -e 'Script must be run as root. Use sudo, su, or add "USER root" to your Dockerfile before running this script.'
    exit 1
fi

export DEBIAN_FRONTEND=noninteractive
check_packages curl ca-certificates apt-transport-https dirmngr gnupg2
curl -sSL "https://dl.google.com/linux/direct/google-chrome-stable_current_$(dpkg --print-architecture).deb" -o /tmp/chrome.deb
apt-get -y install /tmp/chrome.deb
rm -f /tmp/chrome.deb