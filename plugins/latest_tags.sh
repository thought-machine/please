#!/bin/bash

# This script gets the latest tags for each of our first class plugin repos
# It is intended to be run by a github action

accepted_plugins=("python" "java" "go" "cc" "shell")

PLUGIN=$1

if [ -z "$PLUGIN" ]; then
    echo "Usage: $0 <plugin>"
    exit 1
fi

if [[ ! " ${accepted_plugins[@]} " =~ " ${PLUGIN} " ]]; then
    echo "Plugin must be one of: ${accepted_plugins[@]}"
    exit 1
fi

URLPREFIX="https://github.com/please-build/"

# These should be sorted already
LATEST=$(git ls-remote --tags ${URLPREFIX}${PLUGIN}-rules.git | sed 's/.*\///' | tail -n 1)

if [ -z "$LATEST" ]; then
    echo "No tags found for ${PLUGIN}"
    exit 1
fi

echo $LATEST
