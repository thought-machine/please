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
mkdir -p ${DEST}
OUTPUTS="`plz-out/bin/src/please query outputs //package:installed_files`"
for OUTPUT in $OUTPUTS; do
    TARGET="${DEST}/$(basename $OUTPUT)"
    rm -f "$TARGET"  # Important so we don't write through symlinks.
    cp "$OUTPUT" "$TARGET"
    chmod 0775 "$TARGET"
done
ln -sf "${DEST}/please" "${DEST}/plz"
chmod 0664 "${DEST}/junit_runner.jar"

echo "Please installed"

if ! hash plz 2>/dev/null; then
    echo "You might want to run ln -s ~/.please/please /usr/local/bin/plz or add ~/.please to your PATH."
fi
