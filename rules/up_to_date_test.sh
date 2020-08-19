#!/bin/sh

if cmp -s $DATA_BIN_DATA $DATA_GENERATED; then
    echo "Files have the same content"
    exit 0
else
    echo "rules/rules.go is out of date, please run //rules:generate"
    exit 1
fi