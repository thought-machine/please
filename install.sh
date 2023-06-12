#!/usr/bin/env bash
#
# Installs Please and its relevant binaries into ~/.please.
# You must run ./bootstrap.sh before running this.

set -eu

if [ ! -f plz-out/bin/src/please ]; then
    echo "It looks like Please hasn't been built yet."
    echo "Try running ./bootstrap.sh first."
    exit 1
fi

DEST="${HOME}/.please"

# Try to read the local config, if it exists.
if [ -f .plzconfig.local ]; then
    if grep -i '^location' .plzconfig.local > /dev/null; then
        DEST="`grep -i '^location' .plzconfig.local | cut -d '=' -f 2 | tr -d ' '`"
    fi
fi


mkdir -p ${DEST}
OUTPUTS="`plz-out/bin/src/please query outputs //package:installed_files`"
for OUTPUT in $OUTPUTS; do
    TARGET="${DEST}/$(basename $OUTPUT)"
    rm -f "$TARGET"  # Important so we don't write through symlinks.
    cp "$OUTPUT" "$TARGET"
    chmod 0775 "$TARGET"
done
ln -sf "${DEST}/please" "${DEST}/plz"

echo "Please installed into $DEST"

if ! hash plz 2>/dev/null; then
    echo "You might want to run ln -s ~/.please/please /usr/local/bin/plz or add ~/.please to your PATH."
fi
