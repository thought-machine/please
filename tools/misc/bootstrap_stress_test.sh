#!/bin/bash
# Used to stress test caching via bootstrap. Essentially we should bootstrap once to
# build everything, then rerunning it a bunch of times in a loop should be very fast
# and retrieve everything from the cache.
# This script assumes the initial bootstrap has already run.

N="${1:-10}"

for i in `seq 1 $N`; do
    echo "Round $i"
    ./bootstrap.sh --plain_output || { echo "Bootstrap failed"; exit 1; }
    BUILDING="$(grep "Building target" plz-out/log/*.log)"
    if [ -n "$BUILDING" ]; then
        echo "Found building targets:"
        echo "$BUILDING"
        exit 1
    fi
    RUNNING="$(grep "Running test" plz-out/log/test_build.log)"
    if [ -n "$RUNNING" ]; then
        echo "Found running tests:"
        echo "$RUNNING"
        exit 1
    fi
done
