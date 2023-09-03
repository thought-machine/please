#!/usr/bin/env bash

file=$1

if ! test -f "$file"; then
  echo "$file" doesnt exist
  exit 1
fi

CONTENT=$(<"$file")
shift 1
CHECK=$(< <(printf '%s\n' "$@"))

if [[ "$CONTENT" != "$CHECK" ]]; then
  printf '%s\n%s\n%s\n%s\n%s\n' \
    "${file} doesnt contain" \
    "${CHECK}" \
    "---- it contains ----" \
    "${CONTENT}" \
    "---- EOF ----"
  exit 1
fi
