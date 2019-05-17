#!/bin/bash

if [ "`cat $DATA`" != "test/cross_compile/arch_location.txt" ]; then
    echo "Unexpected contents of file: `cat $DATA`"
    exit 1
fi
