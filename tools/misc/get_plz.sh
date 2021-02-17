#!/bin/sh
#
# Downloads a precompiled copy of Please from our s3 bucket and installs it.
set -e

VERSION=`curl -fsSL https://get.please.build/latest_version`
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

PLEASE_URL="https://get.please.build/${GOOS}_amd64/${VERSION}/please_${VERSION}.tar.gz"

LOCATION="${HOME}/.please"
DIR="${LOCATION}/${VERSION}"
mkdir -p "$DIR"

echo "Downloading Please ${VERSION}..."
curl -fsSL "${PLEASE_URL}" | tar -xzpf- --strip-components=1 -C "$DIR"
# Link it all back up a dir
for x in `ls "$DIR"`; do
    ln -sf "${DIR}/${x}" "$LOCATION"
done
ln -sf "${LOCATION}/please" "${LOCATION}/plz"
mkdir "${LOCATION}/bin"
curl https://get.please.build/pleasew -s --output "${LOCATION}/bin/plz"
chmod +x "${LOCATION}/bin/plz"

echo

if ! hash plz 2>/dev/null; then
    if [ -d ~/.local/bin ]; then
        echo "Adding plz to ~/.local/bin..."
        ln -s ~/.please/plz ~/.local/bin/plz
    elif [ -f ~/.profile ]; then
        echo 'export PATH="${PATH}:${HOME}/.please/bin"' >> ~/.profile
        echo "Added Please to path. Run 'source ~/.profile' to pick up the new PATH in this terminal session."
    else
        echo "Unsure how to add to PATH, not modifying anything. If desired, add '~/.please/bin' to your PATH"
        echo "or install please system-wide with 'sudo cp ~/.please/bin/* /usr/bin' or similar"
    fi
fi

echo
echo "Please has been installed under ${LOCATION}"
echo "Run plz --help for more information about how to invoke it,"
echo "or plz help for information on specific help topics."
