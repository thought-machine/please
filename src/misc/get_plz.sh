#!/bin/sh
#
# Downloads a precompiled copy of Please from our s3 bucket and installs it.
set -eu

REVISION=`curl -s https://s3-eu-west-1.amazonaws.com/please-build/latest_version`
# Find the os / arch to download. You can do this quite nicely with go env
# but we use this script on machines that don't necessarily have Go itself.
OS=`uname`
if [ "$OS" = "Linux" ]; then
    GOOS="linux"
elif [ "$OS" = "Darwin" ]; then
    GOOS="darwin"
else
    echo "Unknown operating system $OS"
    exit 1
fi

PLEASE_URL="https://s3-eu-west-1.amazonaws.com/please-build/${GOOS}_amd64/${REVISION}/please.tar.gz"

if [ ! -d /opt/please ]; then
    sudo mkdir -p /opt/please
    sudo chown `whoami` /opt/please
fi

rm -f /opt/please/*[^/]

curl -sSL "${PLEASE_URL}" | tar -zxpf- -C /opt/
ln -sf /opt/please/please /usr/local/bin/plz

echo "Please installed."
plz --help
