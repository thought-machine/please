#!/usr/bin/env bash
set -eu

[ -f "plz-out/please/plz" ] && PLZ="plz-out/please/plz" || PLZ="./pleasew"


$PLZ run //third_party/binary:golangci-lint -p -- run --sort-results src/... tools/...
$PLZ fmt -q || {
    echo "BUILD files are not correctly formatted; run plz fmt -w to fix."
    exit 1
}
