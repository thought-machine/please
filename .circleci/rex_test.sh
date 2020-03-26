#!/usr/bin/env bash

set -eu

trap 'killall elan mettle zeal' SIGINT SIGTERM EXIT

DIR="${1:-~/.please}"

# Extract the plz installation from earlier step
rm -rf "$DIR"
mkdir "$DIR"
tar -xzf /tmp/workspace/please_*.tar.gz --strip-components=1 -C "$DIR"
ln -s "${DIR}/please" "${DIR}/plz"
export PATH="$DIR:$PATH"

# Start the servers in the background
plz run parallel -p -v notice --colour --detach //test/remote:run_elan //test/remote:run_zeal //test/remote:run_mettle

# Test we can rebuild plz itself.
plz build --profile ci_remote -p -v notice --colour //src:please
# Check we can actually run some tests
plz test --profile ci_remote -p -v notice --colour //src/core:all
