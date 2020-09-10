#!/bin/sh

if cmp -s $DATA_BIN_DATA $DATA_GENERATED; then
    echo "Files have the same content"
    exit 0
else
    echo "src/assets/bindata.go is out of date, plz run //src/assets:generate"
    exit 1
fi