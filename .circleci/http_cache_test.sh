#!/usr/bin/env bash

set -eu

trap 'killall http_cache' SIGINT SIGTERM EXIT

DIR="${1:-/tmp/please}"

# Extract the plz installation from earlier step
rm -rf "$DIR"
mkdir "$DIR"
tar -xzf /tmp/workspace/linux_amd64/please_*.tar.gz --strip-components=1 -C "$DIR"

# Start the server in the background
"${DIR}/please" run parallel --env -p -v notice --colour --detach //tools/http_cache:run_local

# Test we can rebuild plz itself.
"${DIR}/please" test --profile localcache -p -v notice --colour //src/...

# Clean out plz-out and the dir cache. This doesn't clean the http cache.
"${DIR}/please" clean -f

# Run that again to make sure cache restoration works
"${DIR}/please" test --profile localcache -p -v notice --colour //src/...
