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
      warn "python3 not found, excluding python3 tests"
      EXCLUDES="${EXCLUDES} --exclude=py3 --exclude python3"
  fi
  if ! hash xz 2>/dev/null ; then
      warn "xz not found, excluding update tests"
      EXCLUDES="${EXCLUDES} --exclude=xz"
  fi
  GCCVER="`cc -dumpversion`"
  if [ ! -d "/usr/lib/gcc/x86_64-linux-gnu/${GCCVER%.*.*}/32" ] && [ ! -d "/usr/lib/gcc/x86_64-pc-linux-gnu/$GCCVER/32" ]; then
      warn "32-bit gcc libraries not found, excluding cross-compile tests"
      EXCLUDES="${EXCLUDES} --exclude=x86"
  fi

  echo $EXCLUDES
}

# Run the tests to make sure they still work
notice "Running tests..."

eval `go env`

# Run the set of tests that will work on this machine. There are a bunch of tests in this
# repo that are optional and exercise specific rules, and require extra dependencies.
EXCLUDES=$(check_path_for_excludes)

plz-out/bin/src/please -p -v2 $PLZ_ARGS ${PLZ_COVER:-test} $EXCLUDES --exclude=e2e --log_file plz-out/log/test_build.log --log_file_level 4 --trace_file plz-out/log/trace.json $@

# We run the end-to-end tests separately to ensure things don't fight with one another; they are
# finicky about some things due to running plz recursively and disabling the lock.
notice "Running end-to-end tests..."
plz-out/bin/src/please -p -v2 $PLZ_ARGS ${PLZ_COVER:-test} $EXCLUDES --include=e2e --log_file plz-out/log/e2e_build.log --log_file_level 4 $@
