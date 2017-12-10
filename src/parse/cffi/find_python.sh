#!/bin/bash
# Script to locate the static libpython.
PKG_CONFIG_NAME="$1"
LIB_FILENAME="$2"
for i in `gcc -print-search-dirs | grep libraries: | cut -c 13- | tr ":" " "` `pkg-config --libs-only-L $PKG_CONFIG_NAME`; do
    if [ -f "${i}/${LIB_FILENAME}" ]; then
	cp "${i}/${LIB_FILENAME}" $OUT
	exit 0
    fi
done

echo "Couldn't find $LIB_FILENAME"
exit 1
