#!/bin/sh
#
# This script implements an example CI flow using plz, which will run all affected tests
# based on files that have changed vs. master.
#
# It performs a two-step test that both builds all possible targets and runs all tests;
# this ensures that any targets that don't have tests are still at least buildable.
#
# It's a bit pessimistic, in that it runs all tests based on any file that has changed,
# assuming all possible modifications (notably this means that if you change a BUILD file
# it must rerun all reverse dependencies of all targets in that file).
# A more advanced flow is possible using something along the lines of:
#   git checkout master
#   plz query graph > master.json
#   git checkout mybranch
#   plz query graph > branch.json
#   git diff --name-only origin/master | \
#       plz tool diff_graphs -- -b master.json -a branch.json - | \
#       plz test -
#
# This performs an exhaustive diff of the build graph which runs all tests that have changed,
# but is not affected by trivial changes to BUILD files and so forth.
# It's a bit more involved though and for many cases this script is easier.

set -eu
set -o pipefail

# Set $ORIGIN to a branch or tag name if you want to build vs. something other than master.
ORIGIN="${ORIGIN:-origin/master}"
echo "\033[32mBuilding targets...\033[0m"
plz query affectedtargets `git diff --name-only $ORIGIN` | plz build -
echo "\033[32mBuilding targets...\033[0m"
plz query affectedtargets --tests `git diff --name-only $ORIGIN` | plz test -
echo "\033[32mDone!\033[0m"
