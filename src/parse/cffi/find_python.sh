#!/bin/bash
# Script to locate the static libpython.
for i in `gcc -print-search-dirs | grep libraries: | cut -c 13- | tr ":" " "` `pkg-config --libs-only-L python2`; do
    if [ -f "$i/libpython2.7.a" ]; then
	cp "$i/libpython2.7.a" $OUT
	exit 0
    fi
done

echo "Couldn't find libpython2.7.a"
exit 1
