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
elif [ "$OS" = "FreeBSD" ]; then
    GOOS="freebsd"
else
    echo "Unknown operating system $OS"
    exit 1
fi

ARCH=`uname -m`
if [ "$ARCH" = "x86_64" ]; then
    ARCH="amd64"
elif [ "$ARCH" = "arm64" ]; then
    :
elif [ "$ARCH" = "aarch64" ]; then
    ARCH="arm64"
else
    echo "Unsupported cpu arch $ARCH"
    exit 1
fi

PLEASE_URL="https://get.please.build/${GOOS}_${ARCH}/${VERSION}/please_${VERSION}.tar.gz"

LOCATION="${HOME}/.please"
DIR="${LOCATION}/${VERSION}"
mkdir -p "$DIR"

echo "Downloading Please ${VERSION}..."
curl -fsSL "${PLEASE_URL}" | tar -xzpf- --strip-components=1 -C "$DIR"
# Link it all back up a dir
for x in "${DIR}/"*; do
    ln -sf "${x}" "$LOCATION"
done

ln -sf "${LOCATION}/please" "${LOCATION}/plz"
mkdir "${LOCATION}/bin"
curl https://get.please.build/pleasew -s --output "${LOCATION}/bin/plz"
chmod +x "${LOCATION}/bin/plz"

if ! hash plz 2>/dev/null; then
    echo
    if [ -d ~/.local/bin ]; then
        echo "Adding plz to ~/.local/bin..."
        ln -s ~/.please/plz ~/.local/bin/plz
    elif [ -f ~/.profile ]; then
        echo 'export PATH="${PATH}:${HOME}/.please/bin"' >> ~/.profile
        echo "Added Please to path. Run 'source ~/.profile' to pick up the new PATH in this terminal session."
    else
        echo "We were unable to automatically add Please to the PATH."
        echo "If desired, add this line to your ~/.profile or equivalent:"
        echo "    'PATH=\${PATH}:~/.please/bin'"
        echo "or install please system-wide with"
        echo "    'sudo cp ~/.please/bin/* /usr/local/bin'"
    fi
fi

echo
echo "Please has been installed under ${LOCATION}"
echo "Run plz --help for more information about how to invoke it,"
echo "or plz help for information on specific help topics."
echo
echo "It is also highly recommended to set up command line completions."
echo "To do so, add this line to your ~/.bashrc or ~/.zshrc:"
echo "    source <(plz --completion_script)"
