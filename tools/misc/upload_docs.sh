#!/usr/bin/env bash
# Script to automatically update the Please website.
# Should be run from the root of the repo, and only by a CI system.

set -eu

BUCKET="s3://please.build"

DIR="`mktemp -d`"
plz-out/bin/src/please export outputs -o "$DIR" //docs
aws s3 cp --recursive "$DIR" "$BUCKET"
rm -rf "$DIR"
