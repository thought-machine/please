#!/bin/bash

if [ "`cat $DATA`" != "wibble" ]; then
    echo "Unexpected contents of file: `cat $DATA`"
    exit 1
fi
