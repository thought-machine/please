#!/usr/bin/env bash
set -eu

[ -f "plz-out/bin/src/please" ] && PLZ="plz-out/bin/src/please" || PLZ="./pleasew"


go list -f '{{.Dir}}' ./src/... ./tools/...  | fgrep -v test_data | xargs \
  $PLZ run //third_party/binary:golangci-lint -p -- run --sort-results
$PLZ fmt -q || {
    echo "BUILD files are not correctly formatted; run plz fmt -w to fix."
    exit 1
}
