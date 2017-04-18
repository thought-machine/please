#!/bin/bash
CONTENTS=`cat test/arch/target_out`
if [[ "$CONTENTS" != "test_amd64" ]]; then
    echo "Incorrect contents: $CONTENTS"
    exit 1
fi
exit 0
