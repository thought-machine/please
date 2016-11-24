#!/bin/bash
#
# Installs Please and its relevant binaries into ~/.please.
# You must run ./bootstrap.sh before running this.

set -eu

if [ ! -f plz-out/bin/src/please ]; then
    echo "It looks like Please hasn't been built yet."
    echo "Try running ./bootstrap.sh first."
    exit 1
fi

DEST="${HOME}/.please"
mkdir -p ${DEST}
rm -f ${DEST}/please ${DEST}/please_pex ${DEST}/junit_runner.jar ${DEST}/jarcat ${DEST}/please_maven ${DEST}/cache_cleaner ${DEST}/*.so
cp -f plz-out/bin/src/please ${DEST}/please
chmod 0775 ${DEST}/please
ln -sf ${DEST}/please ${DEST}/plz
cp -f plz-out/bin/src/libplease_parser_*.so ${DEST}
chmod 0664 ${DEST}/libplease_parser_*.so
cp -f plz-out/bin/src/build/python/please_pex ${DEST}/please_pex
chmod 0775 ${DEST}/please_pex
cp -f plz-out/bin/src/build/java/junit_runner.jar ${DEST}/junit_runner.jar
chmod 0664 ${DEST}/junit_runner.jar
cp -f plz-out/bin/src/build/java/jarcat ${DEST}/jarcat
chmod 0775 ${DEST}/jarcat
cp -f plz-out/bin/src/build/java/please_maven ${DEST}/please_maven
chmod 0775 ${DEST}/please_maven
cp -f plz-out/bin/src/cache/main/cache_cleaner ${DEST}/cache_cleaner
chmod 0775 ${DEST}/cache_cleaner
cp -f plz-out/bin/src/misc/please_diff_graphs ${DEST}/please_diff_graphs
chmod 0775 ${DEST}/please_diff_graphs
cp -f plz-out/bin/src/build/go/please_go_test ${DEST}/please_go_test
chmod 0775 ${DEST}/please_go_test
cp -f plz-out/bin/src/lint/please_build_linter ${DEST}/please_build_linter
chmod 0775 ${DEST}/please_build_linter
cp -f plz-out/bin/src/build/java/build/please/compile/server.jar ${DEST}/javac_worker
chmod 0775 ${DEST}/please_build_linter
echo "Please installed"

if [ ! -f /usr/local/bin/plz ]; then
    echo "You might want to run ln -s ~/.please/please /usr/local/bin/plz or add ~/.please to your PATH."
fi
