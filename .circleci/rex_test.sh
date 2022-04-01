#!/usr/bin/env bash

set -eu

trap 'killall elan mettle zeal' SIGINT SIGTERM EXIT

DIR="/tmp/please"
WORKSPACE_DIR="${1:-/tmp/workspace/linux_amd64}"

# Extract the plz installation from earlier step
rm -rf "$DIR"
mkdir "$DIR"
tar -xzf ${WORKSPACE_DIR}/please_*.tar.gz --strip-components=1 -C "$DIR"
ln -s "${DIR}/please" "${DIR}/plz"
export PATH="$DIR:$PATH"

# Start the servers in the background
echo "Starting servers..."
plz run parallel -p -v notice --colour --detach -o build.passenv:PATH //test/remote:run_elan //test/remote:run_zeal //test/remote:run_mettle

echo "Waiting for servers to come up..."
# Give the servers a chance to start up.
sleep 3

# Test we can rebuild plz itself.
echo "Building please..."
plz build -o build.passenv:PATH --profile ci_remote -p -v notice --colour //src:please

# Check we can actually run some tests
echo "Testing //src/core:all..."
plz test -o build.passenv:PATH --profile ci_remote -p -v notice --colour //src/core:all

# And run any tests we deem to be pertinent to remote execution
echo "Testing anything labeled rex..."
plz test -o build.passenv:PATH --profile ci_remote -p -v notice --colour -i rex
