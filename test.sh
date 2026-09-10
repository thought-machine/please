#!/usr/bin/env bash

set -eu

source ./log.sh

# PLZ_ARGS can be set to pass arguments to all plz invocations in this script.
PLZ_ARGS="${PLZ_ARGS:-}"

# check_path_for_excludes will check the Please path for toolchains and print the relevant exclude args
check_path_for_excludes() {
  # Set the path to match the please path
  PATH=/usr/local/bin:/usr/bin:/bin

  EXCLUDES=""

  if ! hash python3 2>/dev/null ; then
      warn "python3 not found, excluding python tests"
      EXCLUDES="${EXCLUDES} --exclude=py"
  fi
  if ! hash xz 2>/dev/null ; then
      warn "xz not found, excluding update tests"
      EXCLUDES="${EXCLUDES} --exclude=xz"
  fi
  echo $EXCLUDES
}

# has_wine reports whether the Windows tests can run here.
has_wine() {
  hash wine 2>/dev/null
}

# Run the tests to make sure they still work
notice "Running tests..."

eval `go env`

# Run the set of tests that will work on this machine. There are a bunch of tests in this
# repo that are optional and exercise specific rules, and require extra dependencies.
EXCLUDES=$(check_path_for_excludes)

plz-out/bin/src/please -p -v2 $PLZ_ARGS ${PLZ_COVER:-test} $EXCLUDES --exclude=e2e --exclude=wine --log_file plz-out/log/test_build.log --log_file_level 4 --trace_file plz-out/log/trace.json $@

# We run the end-to-end tests separately to ensure things don't fight with one another; they are
# finicky about some things due to running plz recursively and disabling the lock.
notice "Running end-to-end tests..."
plz-out/bin/src/please -p -v2 $PLZ_ARGS ${PLZ_COVER:-test} $EXCLUDES --include=e2e --exclude=wine --log_file plz-out/log/e2e_build.log --log_file_level 4 $@

# The Windows tests cross-compile for windows_amd64 and run the result under Wine. They are a
# third pass because they are the only thing that builds the Go standard library for another
# platform, which is slow and pointless for someone who just wants the unit tests.
if has_wine; then
  notice "Running Windows tests under Wine..."
  plz-out/bin/src/please -p -v2 $PLZ_ARGS ${PLZ_COVER:-test} $EXCLUDES --include=wine --log_file plz-out/log/wine_build.log --log_file_level 4 $@
else
  warn "wine not found, skipping the Windows tests"
fi
