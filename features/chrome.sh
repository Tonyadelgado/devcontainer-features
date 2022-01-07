#!/bin/bash
set -e

# Import common utils
. "$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/scripts/utils.sh"

export DEBIAN_FRONTEND=noninteractive
check_packages curl ca-certificates apt-transport-https dirmngr gnupg2
curl -sSL "https://dl.google.com/linux/direct/google-chrome-stable_current_$(dpkg --print-architecture).deb" -o /tmp/chrome.deb
apt-get -y install /tmp/chrome.deb
rm -f /tmp/chrome.deb