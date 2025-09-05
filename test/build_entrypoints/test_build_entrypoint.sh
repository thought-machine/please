#!/usr/bin/env bash
# This script tests that the Please test name exists in the expected output
# file for an entrypoint.
# This expects the pattern of build rules with Build Entrypoints to output a
# file with the name of their build target into a file with the name of the
# build target.
set -Eeuo pipefail

exec grep "${NAME//_test/}" "${PKG_DIR}/${NAME//_test/}.txt"
