#!/usr/bin/env bash
set -eu

source "test/pass_env.sh"
if [ -z "$USER" ]; then
    echo '$USER is not set'
    exit 1
fi
