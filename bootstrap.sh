#!/bin/bash

set -eu

function notice {
    >&2 echo -e "\033[32m$1\033[0m"
}
function warn {
    >&2 echo -e "\033[33m$1\033[0m"
}
function error {
    >&2 echo -e "\033[31m$1\033[0m"
}

# Fetch the Go dependencies manually
notice "Installing Go dependencies..."
export GOPATH="${PWD}"
go get golang.org/x/crypto/ssh/terminal
go get golang.org/x/sync/errgroup
go get golang.org/x/tools/cover
go get gopkg.in/op/go-logging.v1
go get gopkg.in/gcfg.v1
go get github.com/kevinburke/go-bindata/...
go get github.com/jessevdk/go-flags
go get github.com/dustin/go-humanize
go get github.com/texttheater/golang-levenshtein/levenshtein
go get github.com/Workiva/go-datastructures/queue
go get github.com/coreos/go-semver/semver
go get github.com/djherbis/atime

# Clean out old artifacts.
rm -rf plz-out src/parse/builtin_rules.bindata.go src/parse/asp/builtins/builtin_data.bindata.go
# Compile the builtin rules
notice "Compiling built-in rules..."
go run src/parse/asp/main/compiler.go -o plz-out/asp/src/parse/asp/builtins src/parse/asp/builtins/builtins.build_defs src/parse/rules/*.build_defs
# Embed them into Go
bin/go-bindata -o src/parse/asp/builtins/builtin_data.bindata.go -pkg builtins -prefix plz-out/asp/src/parse/asp/builtins plz-out/asp/src/parse/asp/builtins
# The old version is still needed for things to compile.
bin/go-bindata -o src/parse/builtin_rules.bindata.go -pkg parse -prefix src/parse/rules/ -ignore BUILD src/parse/rules/

# Now invoke Go to run Please to build itself.
notice "Building Please..."
go run src/please.go --plain_output build //src:please --log_file plz-out/log/bootstrap_build.log
# Use it to build the rest of the tools that come with it.
notice "Building the tools..."
plz-out/bin/src/please --plain_output build //package:installed_files --log_file plz-out/log/tools_build.log

if [ $# -gt 0 ] && [ "$1" == "--skip_tests" ]; then
    exit 0
fi

# Run the tests to make sure they still work
notice "Running tests..."

# Run the set of tests that will work on this machine.
# We assume the user has Java installed or the build will have already failed,
# but some other parts are optional until one actually tries to use the rule.
EXCLUDES=""

HAVE_UNITTEST=false
for path in `echo -e | cpp -xc++ -Wp,-v 2>&1 | grep "^ "`; do
    if [ -f "${path}/UnitTest++/UnitTest++.h" ]; then
        HAVE_UNITTEST=true
    fi
done
if ! $HAVE_UNITTEST ; then
    warn "UnitTest++.h not found, excluding C++ tests"
    EXCLUDES="${EXCLUDES} --exclude=cc"
else
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
fi
if ! hash docker 2>/dev/null ; then
    warn "Docker not found, excluding containerised tests"
    EXCLUDES="${EXCLUDES} --exclude=container"
fi
if ! hash python2 2>/dev/null ; then
    warn "python2 not found, excluding python2 tests"
    EXCLUDES="${EXCLUDES} --exclude=py2"
fi
if ! hash python3 2>/dev/null ; then
    warn "python3 not found, excluding python3 tests"
    EXCLUDES="${EXCLUDES} --exclude=py3"
fi
if ! hash pypy 2>/dev/null ; then
    warn "pypy not found, excluding pypy tests"
    EXCLUDES="${EXCLUDES} --exclude=pypy"
fi
if ! hash clang++ 2>/dev/null ; then
    warn "Clang not found, excluding Clang tests"
    EXCLUDES="${EXCLUDES} --exclude=clang"
fi
if ! hash gold 2>/dev/null ; then
    warn "Gold not found, excluding Gold tests"
    EXCLUDES="${EXCLUDES} --exclude=gold"
fi
# If the proto files are installed in a different location, their tests won't work.
if [ ! -d "/usr/include/google/protobuf" ]; then
    warn "google/protobuf not found, excluding relevant tests"
    EXCLUDES="${EXCLUDES} --exclude=proto"
fi

plz-out/bin/src/please test ... $EXCLUDES --log_file plz-out/log/test_build.log --log_file_level 4 $@

# Lint needs python3.
if hash python3 2>/dev/null ; then
    # Don't run this in CI or any unusual workflows.
    if [ $# -eq 0 ] ; then
	tools/misc/ci_lint.py
    fi
fi
