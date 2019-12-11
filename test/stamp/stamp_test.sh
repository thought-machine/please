#!/usr/bin/env bash

unexpected="12345-revision
12345-describe"

if [ "`$DATA`" == "${unexpected}" ]; then
    echo "Stamped variable has not been replaced correctly."
    exit 1
fi
