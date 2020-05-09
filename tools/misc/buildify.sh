#!/usr/bin/env bash

plz="plz-out/bin/src/please"
if [ ! -f $plz ]; then
    plz="./pleasew"
fi

exec $plz run -p -- //third_party/go:buildifier $@ `git ls-files | grep BUILD$`
