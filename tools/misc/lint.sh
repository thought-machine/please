#!/usr/bin/env bash
set -eu

[ -f "plz-out/bin/src/please" ] && PLZ="plz-out/bin/src/please" || PLZ="./pleasew"

$PLZ run //third_party/binary:golangci-lint -p -- run src/... tools/...
$PLZ run //tools/misc:buildify -p -- --mode=check || {
    echo "BUILD files are not correctly formatted; run plz buildify to fix."
    exit 1
}
