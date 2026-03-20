#!/usr/bin/env bash
set -euo pipefail

strict_match=false
exported_file=""
original_file=""
target_name=""

while [[ "$#" -gt 0 ]]; do
  case $1 in
    --strict)
      strict_match=true
      shift
      ;;
    --exported)
      exported_file=$2
      shift 2
      ;;
    --original)
      original_file=$2
      shift 2
      ;;
    --target)
      target_name=$2
      shift 2
      ;;
    -*)
      echo "Unknown option: $1" >&2
      exit 1
      ;;
    *)
      echo "Unknown argument: $1" >&2
      exit 1
      ;;
  esac
done

if [[ -z "$exported_file" ]] || [[ -z "$original_file" ]] || [[ -z "$target_name" ]]; then
  echo "Usage: $0 [--strict] --exported <file> --original <file> --target <name>" >&2
  exit 1
fi

if [[ ! -f "$exported_file" ]]; then
  echo "$exported_file doesnt exist" >&2
  exit 1
fi

if [[ ! -f "$original_file" ]]; then
  echo "$original_file doesnt exist" >&2
  exit 1
fi

readonly START_DELIM="# BUILD_STMT_START"
readonly END_DELIM="# BUILD_STMT_END"

# Using awk to extract statement blocks directly into an array.
blocks=()
while IFS= read -r -d '' block; do
  blocks+=("$block")
done < <(awk -v id="$target_name" -v start_delim="$START_DELIM" -v end_delim="$END_DELIM" '
  BEGIN { in_block = 0; block = "" }
  $0 == start_delim " " id {
    in_block = 1;
    block = "";
    next;
  }
  $0 == end_delim {
    if (in_block) {
      in_block = 0;
      printf "%s\0", block;
    }
    next;
  }
  in_block {
    block = block ? block "\n" $0 : $0
  }
' "$original_file")

if [[ ${#blocks[@]} -eq 0 ]]; then
  echo "Failed to pull original content for $target_name" >&2
  exit 1
fi

# Ensure that ALL required blocks are present in the generated file
for block_content in "${blocks[@]}"; do
  if ! grep -Fq "$block_content" "$exported_file"; then
    printf '%s\n%s\n%s\n%s\n%s\n%s\n' \
      "BUILD statements mismatch" \
      "${exported_file} doesnt contain" \
      "${block_content}" \
      "---- it contains ----" \
      "$(cat "$exported_file")" \
      "---- EOF ----" >&2
    exit 1
  fi
done

# If --strict is enabled, ensure ONLY these blocks are present
# (ignoring all whitespace, newlines and comments).
if [[ "$strict_match" == true ]]; then
  concatenated_blocks=""
  for block_content in "${blocks[@]}"; do
    concatenated_blocks="${concatenated_blocks}${block_content}"
  done

  stripped_blocks=$(echo -n "$concatenated_blocks" | sed 's/#.*//' | tr -d ' \t\n\r')
  stripped_exported_file=$(sed 's/#.*//' "$exported_file" | tr -d ' \t\n\r')

  if [[ "$stripped_blocks" != "$stripped_exported_file" ]]; then
    printf '%s\n' "Strict match failed: exported file contains extra or out-of-order content." >&2
    printf '%s\n' "---- Expected (stripped) ----" >&2
    printf '%s\n' "$stripped_blocks" >&2
    printf '%s\n' "---- Got (stripped) ----" >&2
    printf '%s\n' "$stripped_exported_file" >&2
    exit 1
  fi
fi

exit 0
