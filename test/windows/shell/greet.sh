#!/bin/sh
# The subject of //test/windows:sh_binary_test. Sources a library that only exists because the
# payload was unpacked, and finds it relative to $0, which has to be the file that was run
# rather than this one - the same arrangement as on Unix, where they are one file.
set -eu

lib="$(dirname "$0")/test/windows/shell/lib.sh"
. "$lib"

echo "$GREETING $1"

# Scribble on the unpacked library, so that a second run in the same directory has to replace
# it. Anything the payload leaves behind is a build output and so read-only, and a read-only
# file on Windows cannot be written or replaced at all - which would show up here as either
# this line failing or the next run reading the wrong greeting.
echo 'GREETING="stale"' > "$lib"

exit "$2"
