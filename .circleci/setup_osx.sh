#!/bin/bash

set -eu

# /usr/local/go might get cached.
if [ ! -d "/usr/local/go" ]; then
    curl -fsSL https://dl.google.com/go/go1.23.1.darwin-arm64.tar.gz | sudo tar -xz -C /usr/local
fi
sudo ln -s /usr/local/go/bin/go /usr/local/bin/go

# xz might also.
if ! command -v xz &> /dev/null; then
    brew install xz
fi
sudo ln -s /opt/homebrew/bin/xz /usr/local/bin/xz

# /usr/bin/python3 is Xcode's bundled Python, which is too old for some of our deps.
# Put a newer one in /usr/local/bin, which precedes /usr/bin on plz's default path.
if [ ! -x "/opt/homebrew/opt/python@3.13/bin/python3.13" ]; then
    brew install python@3.13
fi
sudo ln -sf /opt/homebrew/opt/python@3.13/bin/python3.13 /usr/local/bin/python3
