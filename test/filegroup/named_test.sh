#!/bin/bash

WIBBLE=$(cat $DATA_WIBBLE)
WOBBLE=$(cat $DATA_WOBBLE)
if [ "$WIBBLE" != "wibblewibblewibble" ]; then
    echo "Unexpected context in wibble: $WIBBLE"
    exit 1
fi
if [ "$WOBBLE" != "wobblewobblewobble" ]; then
    echo "Unexpected context in wobble: $WOBBLE"
    exit 1
fi
if [ "$DATA_ALL" != "test/filegroup/wibble test/filegroup/wobble" ]; then
    echo "Unexpected context in DATA_ALL: $DATA_ALL"
    exit 1
fi
