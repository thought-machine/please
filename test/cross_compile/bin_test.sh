#!/bin/bash
set -eu

if [ "`test/cross_compile/bin`" != "42" ]; then
    echo "unexpected output"
    exit 1
fi
if [ ! `file test/cross_compile/bin | grep 32-bit` ]; then
    echo "unexpected architecture of binary"
    exit 1
fi
