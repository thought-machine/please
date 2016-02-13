#!/bin/sh
set -eu

URL_BASE=`grep -i "^downloadlocation" .plzconfig | cut -d '=' -f 2 | tr -d ' '`
if [ -z "$URL_BASE" ]; then
    URL_BASE="https://s3-eu-west-1.amazonaws.com/please-build"
fi

VERSION=`grep -i "^version" .plzconfig | cut -d '=' -f 2 | tr -d ' '`
if [ -z "$VERSION" ]; then
    echo "Can't determine version, will use latest."
    VERSION=`curl -s ${URL_BASE}/latest_version`
    echo "Latest is $VERSION"
fi

# We might already have it downloaded...
LOCATION=`grep -i "^location" .plzconfig | cut -d '=' -f 2 | tr -d ' '`
if [ -z "$LOCATION" ]; then
    if [ -z "$HOME" ]; then
	echo "\$HOME not set, not sure where to download to."
	exit 1
    fi
    LOCATION="${HOME}/.please"
fi
TARGET="${LOCATION}/${VERSION}/please"
if [ -f "$TARGET" ]; then
    exec "$TARGET" $@
fi

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
# Don't have any builds other than amd64 at the moment.
ARCH="amd64"

PLEASE_URL="${URL_BASE}/${GOOS}_${ARCH}/${VERSION}/please.tar.gz"
DIR="${LOCATION}/${VERSION}"
echo "Downloading Please ${VERSION} to ${DIR}..."
mkdir -p "$DIR"
curl -fsSL "${PLEASE_URL}" | tar -xzpf- --strip-components=1 -C "$DIR"
exec "$TARGET" $@
