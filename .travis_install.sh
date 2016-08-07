#!/bin/sh
# Provided for Travis, to download dependencies for container infrastructure.
# See https://docs.travis-ci.com/user/migrating-from-legacy/ for details.

set -eu

if [ ! -f "$HOME/protoc" ]; then
    rm -rf "$HOME/protoc"
    echo 'Downloading protobuf...'
    curl -fsSLO https://github.com/google/protobuf/releases/download/v3.0.0/protoc-3.0.0-linux-x86_64.zip
    unzip protoc-3.0.0-linux-x86_64.zip bin/protoc
    mv bin/protoc "$HOME/protoc"
else
    echo 'Using cached protobuf.';
fi
