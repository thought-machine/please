#!/bin/bash

set -eu

if ! hash pypy 2>/dev/null ; then
    echo 'You must have PyPy installed to compile Please.'
    exit 1
fi

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
go get github.com/golang/lint/golint
go get google.golang.org/grpc
go get github.com/kardianos/osext
go get github.com/Songmu/prompter
go get github.com/texttheater/golang-levenshtein/levenshtein

# Clean out old artifacts.
rm -rf plz-out src/parse/cffi/parser_interface.py src/parse/rules/embedded_parser.py
# Generate the cffi compiled source
(cd src/parse/cffi && pypy cffi_compiler.py ../defs.h)
cat src/parse/cffi/parser_interface.py src/parse/cffi/please_parser.py > src/parse/rules/embedded_parser.py
# Invoke this tool to embed the Python scripts.
bin/go-bindata -o src/parse/builtin_rules.go -pkg parse -prefix src/parse/rules/ -ignore BUILD src/parse/rules/
# Similarly for the wrapper script.
bin/go-bindata -o src/utils/wrapper_script.go -pkg utils -prefix src/misc src/misc

# Now invoke Go to run Please to build itself.
echo "Building Please..."
go run src/please.go --plain_output build //src:please --log_file plz-out/log/build.log --log_file_level 4
# Let's prove it works by rebuilding itself with itself.
echo "Rebuilding Please with itself..."
plz-out/bin/src/please --plain_output build //:all_tools //package:tarballs --log_file plz-out/log/build.log --log_file_level 4

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
# Strictly some of these need the library but not the command line tools.
# Pretty sure nobody in the world would be in a situation to care about that distinction though.
if ! hash fstproject 2>/dev/null ; then
    echo "libfst not found, excluding relevant tests"
    EXCLUDES="${EXCLUDES} --exclude=fst"
fi
# If the proto files are installed in a different location, their tests won't work.
if [ ! -d "/usr/include/google/protobuf" ]; then
    echo "google/protobuf not found, excluding relevant tests"
    EXCLUDES="${EXCLUDES} --exclude=proto"
fi

plz-out/bin/src/please test ... --exclude cycle $EXCLUDES --log_file plz-out/log/build.log --log_file_level 4 $@
