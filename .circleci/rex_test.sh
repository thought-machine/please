#!/usr/bin/env bash

set -eu

trap 'killall elan mettle zeal' SIGINT SIGTERM EXIT

DIR="${1:-/tmp/please}"

# Extract the plz installation from earlier step
rm -rf "$DIR"
mkdir "$DIR"
tar -xzf /tmp/workspace/linux_amd64/please_*.tar.gz --strip-components=1 -C "$DIR"
ln -s "${DIR}/please" "${DIR}/plz"
export PATH="$DIR:$PATH"

# Start the servers in the background
plz run parallel -p -v notice --colour --detach -o build.passenv:PATH //test/remote:run_elan //test/remote:run_zeal //test/remote:run_mettle

# Test we can rebuild plz itself.
plz build -o build.passenv:PATH --profile ci_remote -p -v notice --colour //src:please
# Check we can actually run some tests
plz test -o build.passenv:PATH --profile ci_remote -p -v notice --colour //src/core:all
# And run any tests we deem to be pertinent to remote execution
plz test --profile ci_remote -p -v notice --colour -i rex
