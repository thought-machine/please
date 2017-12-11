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

# Maybe we are on OSX. I am unsure how best to locate system frameworks so for now
# just hardcoding this (which presumably will be there most of the time...)
FRAMEWORK="/System/Library/Frameworks/Python.framework/Versions/2.7/lib/python2.7/config/${LIB_FILENAME}"
if [ -f "$FRAMEWORK" ] ; then
    cp "$FRAMEWORK" $OUT
    exit 0
fi

echo "Couldn't find $LIB_FILENAME"
exit 1
