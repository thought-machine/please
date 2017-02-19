#!/bin/sh
# Provided for Travis, to download dependencies for their container
# infrastructure and to set up a custom config.
# See https://docs.travis-ci.com/user/migrating-from-legacy/ for details.

set -eu

cat <<EOF > .plzconfig.local
[build]
path = $HOME:$PATH

[go]
goroot = $GOROOT

[cpp]
cctool = gcc-4.8
cpptool = g++-4.8
defaultoptcflags = --std=c99 -O2 -DNDEBUG
defaultoptcppflags = --std=c++11 -O2 -DNDEBUG

[proto]
protoctool = ${HOME}/protoc

[cache]
dir = $HOME/plz-cache

EOF

if [ ! -f "$HOME/protoc" ]; then
    rm -rf "$HOME/protoc"
    echo 'Downloading protobuf...'
    curl -fsSLO https://github.com/google/protobuf/releases/download/v3.0.0/protoc-3.0.0-linux-x86_64.zip
    unzip protoc-3.0.0-linux-x86_64.zip bin/protoc
    mv bin/protoc "$HOME/protoc"
else
    echo 'Using cached protobuf.';
fi
ln -sf `which python3.5` $HOME/python3
