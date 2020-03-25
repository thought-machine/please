#!/usr/bin/env bash

set -eu

trap 'kill $(jobs -pr)' SIGINT SIGTERM EXIT

# Extract the plz installation from earlier step
mkdir ~/.please
tar -xzf /tmp/workspace/please_*.tar.gz --strip-components=1 -C ~/.please
export PATH="${HOME}/.please:$PATH"

# Start the servers in the background
plz run parallel //test/remote:run_elan //test/remote:run_zeal //test/remote:run_mettle &

# Test we can rebuild plz itself.
plz build --profile ci_remote //src:please
# Check we can actually run some tests
plz test --profile ci_remote //src/core:all
