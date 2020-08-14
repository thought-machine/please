#!/usr/bin/env bash

set -eu

trap 'killall http_cache' SIGINT SIGTERM EXIT

DIR="${1:-/tmp/please}"

# Extract the plz installation from earlier step
rm -rf "$DIR"
mkdir "$DIR"
tar -xzf /tmp/workspace/linux_amd64/please_*.tar.gz --strip-components=1 -C "$DIR"
ln -s "${DIR}/please" "${DIR}/plz"
export PATH="$DIR:$PATH"

# Start the server in the background
plz run -p -v notice --colour --detach //tools/http_cache -- -p 1771 -d plz-out/http_cache

# Test we can rebuild plz itself.
plz test --profile localcache -p -v notice --colour //src/...
