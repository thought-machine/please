#!/usr/bin/env bash
set -eu

[ -f "plz-out/bin/src/please" ] && PLZ="plz-out/bin/src/please" || PLZ="./pleasew"

# SA4006 is too unreliable with too many false positives (sad, it seems useful)
# TODO(peterebden,tatskaari): fix the errcheck issues and enable it.
$PLZ run //third_party/binary:golangci-lint -p -- run -D errcheck -e SA4006 src/... tools/...

$PLZ run //tools/misc:buildify -p -- --mode=check || {
    echo "BUILD files are not correctly formatted; run plz buildify to fix."
    exit 1
}
