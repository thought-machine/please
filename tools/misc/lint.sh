#!/usr/bin/env bash

plz="plz-out/bin/src/please"
BLACKLIST="src/parse/asp/main|//test|tools/please_pex|third_party"

# gofmt and go vet
for TARGET in `$plz query alltargets --include go_src --hidden | grep -v "_test#lib" | grep -v "#main" | grep -v proto | grep -Ev $BLACKLIST`; do
    DIR=`echo $TARGET | cut -f 1 -d ':' | cut -c 3-`
    SRCS=`$plz query print $TARGET -f srcs | grep -v // | while read SRC; do echo $DIR/$SRC; done`
    FILTERED=`plz-out/bin/tools/please_go_filter/please_go_filter -tags bootstrap $SRCS`
    if [ "$FILTERED" != "" ]; then
        go vet -tags bootstrap $FILTERED || {
            echo Failed: go vet -tags bootstrap $FILTERED
            exit 1
        }
        golint --set_exit_status $FILTERED || {
            echo Failed: golint --set_exit_status $FILTERED
            exit 1
        }
        if [ "`gofmt -s -l $FILTERED`" != "" ]; then
            echo "Files are not gofmt'd: gofmt -s -w" $FILTERED
            exit 1
        fi
    fi
done

$plz run //tools/misc:buildify -p -- --mode=check || {
    echo "BUILD files are not correctly formatted; run plz buildify to fix."
    exit 1
}
