#!/usr/bin/env bash

if [ -f test/glob/BUILD ]; then
    echo "Should not be able to glob the BUILD file"
    exit 1
fi

if [ ! -f test/glob/test.txt.symlink ]; then
    echo "Should be able to glob the symlink file"
    exit 1
fi

if [ ! -f test/glob/test.txt ]; then
    echo "Should be able to glob the .txt file"
    exit 1
fi