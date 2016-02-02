#!/bin/bash

# Check this file exists in the expected location.
if [ ! -f src/test/test_data/container_data.txt ]; then
    exit 1
fi
