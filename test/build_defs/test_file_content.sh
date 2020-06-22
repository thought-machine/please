#!/bin/sh

if ! test -f "$1"; then
  echo "$1" doesnt exist
  exit 1
fi

CONTENT=$(cat "$1")

if [ "$CONTENT" != "$2" ]; then
  echo "$1" doesnt contain "$2", it contains "$CONTENT"
  exit 1
fi