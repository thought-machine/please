#!/usr/bin/env bash

if [ "`$DATA`" = "12345" ]; then
    echo "Stamped variable has not been replaced correctly."
    exit 1
fi
