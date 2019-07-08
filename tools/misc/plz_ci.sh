#!/usr/bin/env bash
#
# This script implements an example CI flow using plz, which will perform an exhaustive diff
# of the build graph against master and run all targets that have changed
# This is not affected by trivial changes (e.g. formatting changes to BUILD files).

set -eu
set -o pipefail

# Set $ORIGIN to a branch or tag name if you want to build vs. something other than master.
ORIGIN="${ORIGIN:-origin/master}"
plz query changes --since "$ORIGIN" | plz test -
