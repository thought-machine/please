#!/usr/bin/env bash

gen_file=$1
orig_file=$2
stmt_identifier=$3

if ! test -f "$gen_file"; then
  echo "$gen_file" doesnt exist
  exit 1
fi

if ! test -f "$orig_file"; then
  echo "$orig_file" doesnt exist
  exit 1
fi

# Extract the block into a variable
ORIG_CONTENT=$(sed -n "/# Start BStmt ${stmt_identifier}/,/# End BStmt/{ /# /d; p; }" "$orig_file")

if ! grep -Fq "$ORIG_CONTENT" "$gen_file"; then
  printf '%s\n%s\n%s\n%s\n%s\n%s\n' \
    "BUILD statements mismatch" \
    "${gen_file} doesnt contain" \
    "${ORIG_CONTENT}" \
    "---- it contains ----" \
    "$(cat "$gen_file")" \
    "---- EOF ----"
  exit 1
fi
