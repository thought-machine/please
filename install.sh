#!/bin/bash
#
# Installs Please and its relevant binaries into /opt/please.
# You must run ./bootstrap.sh before running this.

set -eu

[[ -w /opt && -w /usr/local ]] || sudo sh -c "mkdir -p /opt && chown -R $USER /opt /usr/local"

if [ ! -f plz-out/bin/src/please ]; then
    echo "It looks like Please hasn't been built yet."
    echo "Try running ./bootstrap.sh first."
else
    if [ ! -d /opt/please ]; then
        sudo mkdir -p /opt/please
        sudo chown `whoami` /opt/please
    fi
    rm -f /opt/please/please /opt/please/please_pex /opt/please/junit_runner.jar /opt/please/jarcat /opt/please/please_maven /opt/please/cache_cleaner
    cp plz-out/bin/src/please /opt/please/please
    chmod 0775 /opt/please/please
    cp plz-out/bin/src/libplease_parser_*.so /opt/please
    chmod 0664 /opt/please/libplease_parser_*.so
    cp plz-out/bin/src/build/python/please_pex /opt/please/please_pex
    chmod 0775 /opt/please/please_pex
    cp plz-out/bin/src/build/java/junit_runner.jar /opt/please/junit_runner.jar
    chmod 0664 /opt/please/junit_runner.jar
    cp plz-out/bin/src/build/java/jarcat /opt/please/jarcat
    chmod 0775 /opt/please/jarcat
    cp plz-out/bin/src/build/java/please_maven /opt/please/please_maven
    chmod 0775 /opt/please/please_maven
    cp plz-out/bin/src/cache/main/cache_cleaner /opt/please/cache_cleaner
    chmod 0775 /opt/please/cache_cleaner
    cp plz-out/bin/src/misc/plz_diff_graphs /opt/please/please_diff_graphs
    chmod 0775 /opt/please/please_diff_graphs
    cp plz-out/bin/src/build/go/please_go_test /opt/please/please_go_test
    chmod 0775 /opt/please/please_go_test
    cp plz-out/bin/src/lint/plz_build_linter /opt/please/please_build_linter
    chmod 0775 /opt/please/please_build_linter
    ln -sf /opt/please/please /usr/local/bin/plz
    echo "Please installed"
fi
