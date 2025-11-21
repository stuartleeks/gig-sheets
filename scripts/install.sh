#!/bin/bash
set -e

# Expects
# OS=linux or darwin
# ARCH=amd64 or 386

if [[ -z "$OS" ]]; then
echo "$OS"
    echo "OS not set: expected linux or darwin"
    exit 1
fi
if [[ -z "$ARCH" ]]; then
    echo "ARCH not set: expected amd64 or 386"
    exit 1
fi


LATEST_VERSION=$(curl --silent "https://api.github.com/repos/stuartleeks/gig-sheets/releases/latest" | grep -Po '"tag_name": "\K.*?(?=")')
echo $LATEST_VERSION
mkdir -p ~/bin
stripped_version=$(echo $LATEST_VERSION | sed 's/^v//')
TAR_FILE="gigsheets_${stripped_version}_${OS}_${ARCH}.tar.gz"
wget https://github.com/stuartleeks/gig-sheets/releases/download/${LATEST_VERSION}/${TAR_FILE}
tar -C ~/bin -zxvf ${TAR_FILE} gigsheets
chmod +x ~/bin/gigsheets
rm ${TAR_FILE}

