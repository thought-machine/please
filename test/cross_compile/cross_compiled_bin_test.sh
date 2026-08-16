#!/bin/bash
# Checks that a binary cross-compiled for another architecture really is built for it.
# We can't run it here, of course; that it runs at all is covered by bin_test on the host arch.
set -eu

if ! file test/cross_compile/bin | grep -q 'ARM aarch64'; then
    echo "unexpected architecture of binary: `file test/cross_compile/bin`"
    exit 1
fi
