#!/usr/bin/env bash

set -euo pipefail

AUDIT_LOG_DIR="${PWD}/audit_test_dir"

$TOOLS_PLEASE build //:go --audit_log_dir $AUDIT_LOG_DIR

if [ ! -d "$AUDIT_LOG_DIR" ]; then
    echo "ERROR: Audit log dir not created: $AUDIT_LOG_DIR" >&2
    exit 1
fi

DIR_LIST=$(find "$AUDIT_LOG_DIR" -mindepth 1 -maxdepth 1 -type d)
DIR_COUNT=$(echo "$DIR_LIST" | wc -l | xargs)

if [ "$DIR_COUNT" -ne 1 ]; then
    echo "ERROR: Audit log dir contains $DIR_COUNT sub-directories; expected 1" >&2
    exit 1
fi

AUDIT_LOG_DIR_WITH_ID="${AUDIT_LOG_DIR}/$(basename "$DIR_LIST")"


# Check files were created
FILES=(
    "please_invocation.jsonl"
    "build_commands.jsonl"
    "remote_files.jsonl"
)

for FILE_NAME in "${FILES[@]}"; do
    FILE_PATH="${AUDIT_LOG_DIR_WITH_ID}/${FILE_NAME}"
    if [[ ! -f "$FILE_PATH" ]]; then
        echo "ERROR: File not created: $FILE_PATH" >&2
        exit 1
    fi
done

CONTENTS=(
    '"build","//:go"'
    '"build_label":"//:go"'
    '"build_label":"_go#download"'
)

for i in ${!FILES[@]}; do
    FILE_PATH="${AUDIT_LOG_DIR_WITH_ID}/${FILES[$i]}"
    EXPECTED_STR="${CONTENTS[$i]}"
    if ! grep -q "$EXPECTED_STR" "$FILE_PATH"; then
        echo "ERROR: $FILE_PATH does not contain $EXPECTED_STR" >&2
        exit 1
    fi
done

exit 0
