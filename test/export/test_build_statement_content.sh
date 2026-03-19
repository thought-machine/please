#!/usr/bin/env bash


gen_file=$1
if ! test -f "$gen_file"; then
  echo "$gen_file" doesnt exist
  exit 1
fi

# Statement identifier of type path/file:name
stmt_identifier=$2
# string BEFORE the last colon
file_path="${stmt_identifier%:*}"
# string AFTER the last colon
statement_name="${stmt_identifier##*:}"

if ! test -f "$file_path"; then
  echo "$file_path" doesnt exist
  exit 1
fi

# Extract the block into a variable
ORIG_CONTENT=$(sed -n "/# Start BStmt ${statement_name}/,/# End BStmt/{ /# /d; p; }" "$file_path")

if ! grep -Fq "$ORIG_CONTENT" "$gen_file"; then
  printf '%s\n%s\n%s\n%s\n%s\n' \
    "${gen_file} doesnt contain" \
    "${ORIG_CONTENT}" \
    "---- it contains ----" \
    "$(cat "$gen_file")" \
    "---- EOF ----"
  exit 1
fi
