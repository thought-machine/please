#!/bin/bash

# /usr/local/go might get cached.
if [ ! -d "/usr/local/go" ]; then
    curl -fsSL https://dl.google.com/go/go1.11.5.darwin-amd64.tar.gz | sudo tar -xz -C /usr/local
fi
sudo ln -s /usr/local/go/bin/go /usr/local/bin/go
