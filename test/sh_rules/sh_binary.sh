#!/usr/bin/env bash
set -eu

# test /bin/bash can be used
[ ! -z "$BASH_VERSION" ] || (echo "Expected a bash shell" >&2 && exit 11)

source test/sh_rules/sh_lib.sh
[ "$TEST_DEPS_VAR" = "123" ] || (echo "Could not source variable from dependency" && exit 12)
