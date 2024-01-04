#!/bin/bash

set -eu

# /usr/local/go might get cached.
if [ ! -d "/usr/local/go" ]; then
    curl -fsSL https://dl.google.com/go/go1.21.5.darwin-amd64.tar.gz | sudo tar -xz -C /usr/local
fi
sudo ln -s /usr/local/go/bin/go /usr/local/bin/go

# xz might also.
if [ ! -f "/usr/local/bin/xz" ]; then
    curl -fsSL https://get.please.build/third_party/binary/xz-5.2.4-darwin_amd64 -o /usr/local/bin/xz
    chmod +x /usr/local/bin/xz
fi
