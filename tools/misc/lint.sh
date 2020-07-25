#!/usr/bin/env bash
set -eu

# SA4006 is too unreliable with too many false positives (sad, it seems useful)
# TODO(peterebden,tatskaari): fix the errcheck issues and enable it.
golangci-lint run -D errcheck -e SA4006 src/... tools/...

[ -f "plz-out/bin/src/please" ] && PLZ="plz-out/bin/src/please" || PLZ="./pleasew"
$PLZ run //tools/misc:buildify -p -- --mode=check || {
    echo "BUILD files are not correctly formatted; run plz buildify to fix."
    exit 1
}
