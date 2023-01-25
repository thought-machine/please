#!/bin/bash

# This script gets the latest tags for each of our first class plugin repos
# It is intended to be run by a github action

plugins=("python" "java" "go" "cc" "shell")

URLPREFIX="https://github.com/please-build/"

# loop through plugins and check version
for PLUGIN in "${plugins[@]}"; do
    LATEST=$(git ls-remote --tags ${URLPREFIX}${PLUGIN}-rules.git | sed 's/.*\///' | tail -n 1)
    if [ -z "$LATEST" ]; then
        echo "No tags found for ${PLUGIN}"
        exit 1
    fi

    DOCS_VERSION=$(./pleasew query print //docs/... --include ${PLUGIN}_plugin_docs* | grep labels | cut -d: -f2 | cut -d\' -f1)
    if [ -z "$DOCS_VERSION" ]; then
        echo "No docs found for ${PLUGIN}"
        exit 1
    fi

    if [ "$LATEST" != "$DOCS_VERSION" ]; then
        echo "Latest version for ${PLUGIN} is ${LATEST}, update the plugin version in docs/BUILD from ${DOCS_VERSION} to ${LATEST}"
        exit 1
    fi
done
