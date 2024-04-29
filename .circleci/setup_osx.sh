#!/bin/bash

set -eu

# /usr/local/go might get cached.
if [ ! -d "/usr/local/go" ]; then
    curl -fsSL https://dl.google.com/go/go1.21.5.darwin-arm64.tar.gz | sudo tar -xz -C /usr/local
fi
sudo ln -s /usr/local/go/bin/go /usr/local/bin/go

# xz might also.
if ! command -v xz &> /dev/null; then
    brew install xz
    ln -s /opt/homebrew/bin/xz /usr/local/bin/xz
fi
