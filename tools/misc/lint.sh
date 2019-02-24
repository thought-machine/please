#!/usr/bin/env bash

alias plz="plz-out/bin/src/please"
BLACKLIST="src/parse/asp/main|tools/cache|//test|tools/please_pex"

for TARGET in `plz query alltargets --include go_src --hidden | grep -v "_test#lib" | grep -v proto | grep -Ev $BLACKLIST`; do
    DIR=`echo $TARGET | cut -f 1 -d ':' | cut -c 3-`
    SRCS=`plz query print $TARGET -f srcs | grep -v // | while read SRC; do echo $DIR/$SRC; done`
    FILTERED=`plz-out/bin/tools/please_go_filter/please_go_filter -tags bootstrap $SRCS`
    if [ "$FILTERED" != "" ]; then
        go vet -tags bootstrap $FILTERED || {
            echo Failed: go vet -tags bootstrap $FILTERED
            exit 1;
        }
        if [ "`gofmt -s -l $FILTERED`" != "" ]; then
            echo "Files are not gofmt'd: gofmt -s -l" $FILTERED
            exit 1;
        fi
    fi
done

plz run //tools/misc:buildify -p -- --mode=check || exit 1
