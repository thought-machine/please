#!/bin/bash
# Script to locate the static libpython.
PKG_CONFIG_NAME="$1"
LIB_FILENAME="$2"
SEARCH_DIRS="`gcc -print-search-dirs | grep libraries: | cut -c 13- | tr ':' ' '`"
PC_DIR="`pkg-config --static --libs-only-L $PKG_CONFIG_NAME | sed -e 's|^-L||g'`"
for i in $SEARCH_DIRS $PC_DIR "${PC_DIR}/${PKG_CONFIG_NAME/-/}/config"; do
    if [ -f "${i}/${LIB_FILENAME}" ]; then
        cp "${i}/${LIB_FILENAME}" $OUT
        exit 0
    fi
done

echo "Couldn't find $LIB_FILENAME"
exit 1
