#!/bin/sh

file=$1

if ! test -f "$file"; then
  echo "$file" doesnt exist
  exit 1
fi

CONTENT=$(cat "$file")
shift 1

if [ "$CONTENT" != "$@" ]; then
  echo "$file" doesnt contain "$@", it contains "$CONTENT"
  exit 1
fi
