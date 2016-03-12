#!/bin/sh
# Provided for Travis, to download dependencies for container infrastructure.
# See https://docs.travis-ci.com/user/migrating-from-legacy/ for details.

set -eu

if [ ! -d "$HOME/protobuf/lib" ]; then
    echo 'Downloading protobuf...'
    curl -fsSL https://github.com/google/protobuf/archive/v3.0.0-beta-2.tar.gz | tar -xz
    (
	cd protobuf-3.0.0-beta-2
	./autogen.sh
	./configure --prefix=$HOME/protobuf
	make -j2
	make install
    )
else
    echo 'Using cached protobuf.';
fi

if [ ! -d "$HOME/pypy/bin/pypy" ]; then
    echo 'Downloading pypy...'
    mkdir $HOME/pypy
    curl -fsSL https://bitbucket.org/pypy/pypy/downloads/pypy-5.0.0-linux64.tar.bz2 | tar -xj --strip-components=1 -C $HOME/pypy
else
    echo 'Using cached pypy.';
fi
