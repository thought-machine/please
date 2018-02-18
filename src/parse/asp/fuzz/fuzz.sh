#!/bin/bash
# This script should be run from the repo root.
set -eu

function notice {
    >&2 echo -e "\033[32m$1\033[0m"
}

notice "Setting up fuzzing corpus..."
TARGET="$(plz query outputs -p //src/parse/asp/fuzz:build)"
WORKDIR="$(dirname $TARGET)"
CORPUS="${WORKDIR}/corpus"
rm -rf "$WORKDIR/crashers"
mkdir -p "$CORPUS"
for FILE in `git ls-files | grep -E "(BUILD|build_defs)"`; do
    cp "$FILE" "${CORPUS}/$(echo $FILE | tr '/' '_')"
done

notice "Instrumenting fuzzer binary..."
plz-out/bin/src/please build //src/parse/asp/fuzz:build
notice "Beginning fuzzing. Work dir: $WORKDIR Binary: $TARGET"
plz-out/bin/src/please run //third_party/go:go-fuzz -- -bin "$TARGET" -workdir="$WORKDIR" -minimize 20s
