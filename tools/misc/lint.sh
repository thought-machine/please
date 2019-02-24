#!/usr/bin/env bash

alias plz="plz-out/bin/src/please"
BLACKLIST="src/parse/asp/main|tools/cache|//test"

for TARGET in `plz query alltargets --include go_src --hidden | grep -v "_test#lib" | grep -v proto | grep -Ev $BLACKLIST`; do
    DIR=`echo $TARGET | cut -f 1 -d ':' | cut -c 3-`
    SRCS=`plz query input $TARGET | grep -E "$DIR/[^/]+\.go" | grep -v plz-out`
    FILTERED=`plz-out/bin/tools/please_go_filter/please_go_filter -tags bootstrap $SRCS`
    if [ "$FILTERED" != "" ]; then
        go vet -tags bootstrap $FILTERED || {
            echo Failed: go vet -tags bootstrap $FILTERED
            exit 1;
        }
    fi
done

# for DIR in `git ls-files | grep "\.go" | grep -v _test.go | grep -v test_data | xargs dirname | sort | uniq`; do
#     go vet -tags bootstrap $DIR/*.go
# done
