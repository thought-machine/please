#!/bin/bash

# This script gets the latest tags for each of our first class plugin repos
# It is intended to be run by a github action

PLUGINS=("python" "java" "go" "cc" "shell" "go-proto" "proto")
URL_PREFIX="https://github.com/please-build/"

failed=0
for plugin in "${PLUGINS[@]}"; do
    latest=$(git ls-remote --tags "${URL_PREFIX}""${plugin}"-rules.git | sed 's/.*\///' | sed '/^v[0-9]\+\.[0-9]\+\.[0-9]\+$/!d' | tail -n 1)
    if [ -z "$latest" ]; then
        echo "No tags found for ${plugin}"
        exit 1
    fi

    docs_version=$(./pleasew query print //docs/... --include "${plugin}"_plugin_docs* | grep labels | cut -d: -f2 | cut -d\' -f1)
    if [ -z "$docs_version" ]; then
        echo "No docs found for ${plugin}"
        exit 1
    fi

    if [ "$latest" != "$docs_version" ]; then
        echo "Latest version for ${plugin} is ${latest}, update the plugin version in docs/BUILD from ${docs_version} to ${latest}"
        failed=1
        continue
    fi

    echo "Latest version for ${plugin} is ${latest}. Docs are up to date"
done

if [ $failed -eq 1 ]; then
    exit 1
fi
