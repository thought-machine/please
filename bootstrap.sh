#!/usr/bin/env bash

set -eu

function notice {
    >&2 echo -e "\033[32m$1\033[0m"
}
function warn {
    >&2 echo -e "\033[33m$1\033[0m"
}

# PLZ_ARGS can be set to pass arguments to all plz invocations in this script.
PLZ_ARGS="${PLZ_ARGS:-}"

# Now invoke Go to run Please to build itself.
notice "Bootstrapping please..."
go build -o plz-out/bootstrap/please_go tools/please_go/please_go.go
go run src/please.go $PLZ_ARGS --log_file plz-out/log/bootstrap_build.log -o please.location:"$(pwd)/plz-out/bootstrap" export outputs -o plz-out/please //package:installed_files
cp -f plz-out/please/please plz-out/please/plz

echo "Please has been installed under plz-out/please"

if [ $# -gt 0 ] && [ "$1" == "--skip_tests" ]; then
    echo "Skipping tests... done."
    exit 0
fi

# We need to build the tools again because please.location has changed, so the e2e test would fail
notice "Building the tools..."
plz-out/please/plz $PLZ_ARGS build //package:installed_files --log_file plz-out/log/tools_build.log

# Run the tests to make sure they still work
notice "Running tests..."

eval `go env`

# check_path_for_excludes will check the Please path for toolchains and print the relevant exclude args
check_path_for_excludes() {
  # Set the path to match the please path
  PATH=/usr/local/bin:/usr/bin:/bin

  EXCLUDES=""
  if [ "`uname`" = "Darwin" ]; then
      if ! hash nasm 2>/dev/null ; then
          # OSX comes with an ancient version of nasm that can't target
          # 64-bit Mach-O binaries (?!!). Ensure we've got the Brew one.
          if [ -n "`nasm -v | grep 'version 2'`" ]; then
              warn "nasm 2.x not found, excluding C++ tests"
              EXCLUDES="${EXCLUDES} --exclude=cc"
          fi
      fi
  fi

  if [ "$GOOS" != "linux" ] ; then
      warn "cc_module tests disabled due to not being on Linux"
      EXCLUDES="${EXCLUDES} --exclude=cc_module"
  fi
  if ! hash python2 2>/dev/null ; then
      warn "python2 not found, excluding python2 tests"
      EXCLUDES="${EXCLUDES} --exclude=py2"
  fi
  if ! hash python3 2>/dev/null ; then
      warn "python3 not found, excluding python3 tests"
      EXCLUDES="${EXCLUDES} --exclude=py3 --exclude python3"
  fi
  if ! pkg-config python3 2>/dev/null ; then
      warn "python3 includes not found, excluding py3 API tests"
      EXCLUDES="${EXCLUDES} --exclude=py3_pkg_config"
  fi
  if ! hash clang++ 2>/dev/null ; then
      warn "Clang not found, excluding Clang tests"
      EXCLUDES="${EXCLUDES} --exclude=clang"
  fi
  if ! hash gold 2>/dev/null ; then
      warn "Gold not found, excluding Gold tests"
      EXCLUDES="${EXCLUDES} --exclude=gold"
  fi
  if ! hash java 2>/dev/null ; then
      warn "Java not found, excluding Java tests"
      EXCLUDES="${EXCLUDES} --exclude=java"
  elif [ "`uname`" = "Darwin" ]; then
      if [ -n "$(find /Library/Java/JavaVirtualMachines -prune -empty)" ] ; then
          warn "JVM not found, excluding Java tests"
          EXCLUDES="${EXCLUDES} --exclude=java"
      fi
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

# Run the set of tests that will work on this machine. There are a bunch of tests in this
# repo that are optional and exercise specific rules, and require extra dependencies.
EXCLUDES=$(check_path_for_excludes)

plz-out/please/plz $PLZ_ARGS ${PLZ_COVER:-test} $EXCLUDES --exclude=e2e --log_file plz-out/log/test_build.log --log_file_level 4 --trace_file plz-out/log/trace.json $@

# We run the end-to-end tests separately to ensure things don't fight with one another; they are
# finicky about some things due to running plz recursively and disabling the lock.
notice "Running end-to-end tests..."
plz-out/please/plz $PLZ_ARGS ${PLZ_COVER:-test} $EXCLUDES --include=e2e --log_file plz-out/log/e2e_build.log --log_file_level 4 $@
