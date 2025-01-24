#!/usr/bin/env bash

set -eu

source ./log.sh

# PLZ_ARGS can be set to pass arguments to all plz invocations in this script.
PLZ_ARGS="${PLZ_ARGS:-}"

# Now invoke Go to run Please to build itself.
notice "Bootstrapping please..."
go run -race src/please.go -p -v2 $PLZ_ARGS --log_file plz-out/log/bootstrap_build.log build //src:please

if [ $# -gt 0 ] && [ "$1" == "--skip_tests" ]; then
    notice "Skipping tests... done."
    exit 0
fi

exec ./test.sh $@