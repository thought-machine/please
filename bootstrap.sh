#!/bin/bash

set -eu

function interpreter_target {
    if hash $1 2>/dev/null ; then
	if [ ! `$1 -c 'import cffi'` ]; then
	    echo " //src:please_parser_$1"
	else
	    >&2 echo "$1 doesn't have cffi installed, won't build parser engine for it."
            >&2 echo "You won't be able to build Please packages unless all parsers are present."
	fi
    else
        >&2 echo "$1 not found; won't build parser engine for it."
        >&2 echo "You won't be able to build Please packages unless all parsers are present."
    fi
}

function interpreter {
    if hash $1 2>/dev/null ; then
	    echo "$1"
    fi
}

# Fetch the Go dependencies manually
echo "Installing Go dependencies..."
export GOPATH="${PWD}"
go get golang.org/x/crypto/ssh/terminal
go get golang.org/x/tools/cover
go get gopkg.in/op/go-logging.v1
go get gopkg.in/gcfg.v1
go get github.com/jteeuwen/go-bindata/...
go get github.com/jessevdk/go-flags
go get github.com/dustin/go-humanize
go get github.com/kardianos/osext
go get github.com/Songmu/prompter
go get github.com/texttheater/golang-levenshtein/levenshtein
go get github.com/Workiva/go-datastructures/queue
go get github.com/fsnotify/fsnotify

# Determine which interpreter engines we'll build.
INTERPRETERS="$(interpreter_target pypy)$(interpreter_target python2)$(interpreter_target python3)"
if [ -z "$INTERPRETERS" ]; then
    echo "No known Python interpreters found, can't build parser engine"
    exit 1
fi
# Choose one to build with during bootstrap
PYPY="$(interpreter pypy)"
PYTHON2="$(interpreter python2)"
PYTHON3="$(interpreter python3)"
INTERPRETER="${PYPY:-${PYTHON2:-${PYTHON3}}}"

# Clean out old artifacts.
rm -rf plz-out src/parse/cffi/parser_interface.py src/parse/rules/embedded_parser.py
# Generate the cffi compiled source
(cd src/parse/cffi && $INTERPRETER cffi_compiler.py defs.h please_parser.py)
# Invoke this tool to embed the Python scripts.
bin/go-bindata -o src/parse/builtin_rules.go -pkg parse -prefix src/parse/rules/ -ignore BUILD src/parse/rules/
# Similarly for the wrapper script.
bin/go-bindata -o src/utils/wrapper_script.go -pkg utils -prefix src/misc src/misc

# Now invoke Go to run Please to build itself.
echo "Building Please..."
SCRIPT_DIR=$(cd "$(dirname "$0")"; pwd)
ENGINE="${SCRIPT_DIR}/src/parse/cffi/libplease_parser_${INTERPRETER}.*"
go run src/please.go --engine $ENGINE --plain_output build //src:please $INTERPRETERS --log_file plz-out/log/bootstrap_build.log --log_file_level 4
# Use it to build the rest of the tools that come with it.
# NB. We can't do the tarballs here because they depend on all the interpreters, which some
#     users might not have installed.
echo "Building the tools..."
plz-out/bin/src/please --plain_output build //src:please //:all_tools --log_file plz-out/log/tools_build.log --log_file_level 4

if [ $# -gt 0 ] && [ "$1" == "--skip_tests" ]; then
    exit 0
fi

# Run the tests to make sure they still work
echo "Running tests..."

# Run the set of tests that will work on this machine.
# We assume the user has Java and Python installed or the build will have already failed,
# but some other parts are optional until one actually tries to use the rule.
EXCLUDES=""

HAVE_UNITTEST=false
for path in `echo | cpp -xc++ -Wp,-v 2>&1 | grep "^ "`; do
    if [ -f "${path}/unittest++/UnitTest++.h" ]; then
        HAVE_UNITTEST=true
    fi
done
if ! $HAVE_UNITTEST ; then
    echo "UnitTest++.h not found, excluding C++ tests"
    EXCLUDES="${EXCLUDES} --exclude=cc"
fi
if ! hash docker 2>/dev/null ; then
    echo "Docker not found, excluding containerised tests"
    EXCLUDES="${EXCLUDES} --exclude=container"
fi
if ! hash python3 2>/dev/null ; then
    echo "python3 not found, excluding python3 tests"
    EXCLUDES="${EXCLUDES} --exclude=py3"
fi
if ! hash clang++ 2>/dev/null ; then
    echo "Clang not found, excluding Clang tests"
    EXCLUDES="${EXCLUDES} --exclude=clang"
fi
if ! hash gold 2>/dev/null ; then
    echo "Gold not found, excluding Gold tests"
    EXCLUDES="${EXCLUDES} --exclude=gold"
fi
# If the proto files are installed in a different location, their tests won't work.
if [ ! -d "/usr/include/google/protobuf" ]; then
    echo "google/protobuf not found, excluding relevant tests"
    EXCLUDES="${EXCLUDES} --exclude=proto"
fi

plz-out/bin/src/please test ... --exclude cycle $EXCLUDES --log_file plz-out/log/test_build.log --log_file_level 4 $@
