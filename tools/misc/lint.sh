#!/usr/bin/env bash
set -eu

go vet github.com/thought-machine/please/src/... github.com/thought-machine/please/tools/...
golint src/... tools/...
if [ "`find src/ tools/ -name '*.go' | xargs gofmt -s -l `" != "" ]; then
    echo "Files are not gofmt'd: find src/ tools/ -name '*.go' | xargs gofmt -s -w"
    exit 1
fi
# SA4006 is too unreliable with too many false positives (sad, it seems useful)
# TODO(peterebden,tatskaari): fix the errcheck issues and enable it.
golangci-lint run -D errcheck -e SA4006 src/... tools/...

[ -f "plz-out/bin/src/please" ] && PLZ="plz-out/bin/src/please" || PLZ="./pleasew"
$PLZ run //tools/misc:buildify -p -- --mode=check || {
    echo "BUILD files are not correctly formatted; run plz buildify to fix."
    exit 1
}
