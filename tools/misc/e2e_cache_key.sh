#!/usr/bin/env bash
#
# Prints the contents of every file that determines what ends up in the e2e tests' shared dir cache
# (see SHARED_CACHE_DIR in //test/build_defs:test.build_defs), for CI to checksum as a cache key.
#
# In practice that's the plugin revisions and Go toolchains the test repos pin: those account for
# almost all of the cache, and everything else in it is cheap enough to rebuild. Entries are
# content-addressed, so a key that's too coarse only costs us some dead weight in the cache, which
# the dir cache's own water marks take care of.
#
# This globs rather than listing files so that a newly added test repo is picked up automatically.

set -euo pipefail

cd "$(dirname "$0")/../.."

# cksum prints the checksum, the size and the name, so a file being renamed changes the output too.
# Sorted so the ordering doesn't depend on how the filesystem hands them back.
find test \( -path '*/plugins/BUILD_FILE' -o -path '*/third_party/go/BUILD_FILE' \) -type f \
    | LC_ALL=C sort \
    | xargs cksum
