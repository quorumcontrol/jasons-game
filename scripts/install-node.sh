#!/bin/bash
set -eo pipefail

apt-get update
apt-get install -y lsb-release apt-transport-https

apt-key add scripts/checksums/nodesource.gpg.key
VERSION=node_12.x
DISTRO="$(lsb_release -s -c)"
echo "deb https://deb.nodesource.com/$VERSION $DISTRO main" | tee /etc/apt/sources.list.d/nodesource.list
echo "deb-src https://deb.nodesource.com/$VERSION $DISTRO main" | tee -a /etc/apt/sources.list.d/nodesource.list

apt-get update
apt-get install -y nodejs
