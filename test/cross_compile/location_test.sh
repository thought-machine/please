#!/bin/bash
# $DATA is arch_location.txt, whose contents are the $(location) of :record_arch. That must come
# out the same whether or not we're cross-compiling, which is what #582 was about.

if [ "`cat $DATA`" != "test/cross_compile/arch.txt" ]; then
    echo "Unexpected contents of file: `cat $DATA`"
    exit 1
fi
