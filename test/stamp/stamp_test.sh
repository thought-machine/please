#!/usr/bin/env bash

git_revision=$(git rev-parse HEAD)
git_describe=$(git describe --always)

expected="${git_revision}
${git_describe}"

if [ "`$DATA`" != "${expected}" ]; then
    echo "`${DATA}` is not equal to ${expected}"
    echo "Stamped variable has not been replaced correctly."
    exit 1
fi
