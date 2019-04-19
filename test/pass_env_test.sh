#!/usr/bin/env bash
set -eu

source "test/pass_env.sh"
if [ -z "$SHELL" ]; then
    echo '$SHELL is not set'
    exit 1
fi
