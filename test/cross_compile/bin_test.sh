#!/bin/bash
# Checks that the binary built for the host architecture actually works, so that the
# cross-compiled test below is checking the architecture of something that isn't broken.
set -eu

if [ "`test/cross_compile/bin`" != "42" ]; then
    echo "unexpected output: `test/cross_compile/bin`"
    exit 1
fi
