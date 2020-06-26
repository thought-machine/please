#!/usr/bin/env bash

EXPECTED="test/filegroup/hash_filegroup_test-c340e9699046999e0dead1f677a889fc94d90488d3a524183afd30b4e17f80aa.txt"
if [ "$DATA" != "$EXPECTED" ]; then
    echo "Unexpected hash filegroup name; was $DATA, should be $EXPECTED"
    exit 1
fi
