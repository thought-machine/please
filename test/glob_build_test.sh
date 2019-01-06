#!/usr/bin/env bash

if [ -f test/BUILD ]; then
    echo "Should not be able to glob the BUILD file"
    exit 1
fi
