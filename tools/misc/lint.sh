#!/usr/bin/env bash
set -eu

[ -f "plz-out/bin/src/please" ] && PLZ="plz-out/bin/src/please" || PLZ="./pleasew"

$PLZ run //third_party/binary:golangci-lint -p -- run --skip-dirs test_data --sort-results src/... tools/...
$PLZ fmt -q || {
    echo "BUILD files are not correctly formatted; run plz fmt -w to fix."
    exit 1
}
