#!/bin/sh
# Provided for Travis, to download dependencies for container infrastructure.
# See https://docs.travis-ci.com/user/migrating-from-legacy/ for details.

set -eu

if [ ! -f "$HOME/protoc" ]; then
    echo 'Downloading protobuf...'
    curl -fsSLO https://github.com/google/protobuf/releases/download/v3.0.0-beta-2/protoc-3.0.0-beta-2-linux-x86_64.zip
    unzip protoc-3.0.0-beta-2-linux-x86_64.zip protoc
    mv protoc $HOME/protoc
else
    echo 'Using cached protobuf.';
fi

if [ ! -f "$HOME/pypy/bin/pypy" ]; then
    rm -rf $HOME/pypy
    echo 'Downloading pypy...'
    mkdir $HOME/pypy
    curl -fsSL https://bitbucket.org/pypy/pypy/downloads/pypy-5.0.0-linux64.tar.bz2 | tar -xj --strip-components=1 -C $HOME/pypy
else
    echo 'Using cached pypy.';
fi
