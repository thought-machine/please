#!/bin/sh

# Check this file exists in the expected location.
if [ ! -f test/container_data.txt ]; then
    exit 1
fi
