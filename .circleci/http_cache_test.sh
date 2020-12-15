#!/usr/bin/env bash

set -eu

trap 'killall http_cache' SIGINT SIGTERM EXIT

DIR="${1:-/tmp/please}"

# Extract the plz installation from earlier step
rm -rf "$DIR"
mkdir "$DIR"
tar -xzf /tmp/workspace/linux_amd64/please_*.tar.xz --strip-components=1 -C "$DIR"
ln -s "${DIR}/please" "${DIR}/plz"
export PATH="$DIR:$PATH"

# Start the server in the background
plz run parallel -p -v notice --colour --detach //tools/http_cache:run_local

# Test we can rebuild plz itself.
plz test --profile localcache -p -v notice --colour //src/...

# Clean out plz-out and the dir cache. This doesn't clean the http cache.
plz clean -f

# Run that again to make sure cache restoration works
plz test --profile localcache -p -v notice --colour //src/...
