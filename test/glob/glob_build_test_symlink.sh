#!/usr/bin/env bash

if [ -f test/glob/test.txt.symlink ]; then
    echo "Shouldn't be able to glob the symlink file"
    exit 1
fi
