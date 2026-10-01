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

# Check that a target's runtime data gets downloaded along with it.
# src/plz builds runtime & data deps in parallel with the target itself, so
# //test/remote_data:slow_data deliberately takes longer to build than the target that
# depends on it. Both are named explicitly with --rebuild so that slow_data really does
# execute rather than being served from the action cache, which would let it win the race.
echo "Testing download of runtime data..."
rm -rf plz-out/gen/test/remote_data
plz build -o build.passenv:PATH --profile ci_remote -p -v notice --colour --rebuild \
    //test/remote_data:needs_data //test/remote_data:slow_data
if [ ! -f plz-out/gen/test/remote_data/slow_data.txt ]; then
    echo "Runtime data for //test/remote_data:needs_data wasn't downloaded"
    exit 1
fi
