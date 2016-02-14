#!/bin/sh
set -eu

# We might already have it downloaded...
LOCATION=`grep -i "^location" .plzconfig | cut -d '=' -f 2 | tr -d ' '`
if [ -z "$LOCATION" ]; then
    if [ -z "$HOME" ]; then
	echo "\$HOME not set, not sure where to look for Please."
	exit 1
    fi
    LOCATION="${HOME}/.please"
fi
# If this exists at any version, let it handle any update.
TARGET="${LOCATION}/please"
if [ -f "$TARGET" ]; then
    exec "$TARGET" $@
fi

URL_BASE=`grep -i "^downloadlocation" .plzconfig | cut -d '=' -f 2 | tr -d ' '`
if [ -z "$URL_BASE" ]; then
    URL_BASE="https://s3-eu-west-1.amazonaws.com/please-build"
fi

VERSION=`grep -i "^version" .plzconfig | cut -d '=' -f 2 | tr -d ' '`
if [ -z "$VERSION" ]; then
    echo "Can't determine version, will use latest."
    VERSION=`curl -fsSL ${URL_BASE}/latest_version`
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
# Potentially we could reuse this but it's easier not to really.
if [ ! -d "$DIR" ]; then
    rm -rf "$DIR"
fi
echo "Downloading Please ${VERSION} to ${DIR}..."
mkdir -p "$DIR"
curl -fsSL "${PLEASE_URL}" | tar -xzpf- --strip-components=1 -C "$DIR"
# Link it all back up a dir
for x in `ls "$DIR"`; do
    ln -sf "${DIR}/${x}" "$LOCATION"
done
ln -sf "${DIR}/please" "${LOCATION}/plz"
exec "$TARGET" $@
