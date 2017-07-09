#!/bin/bash
# Script to extract __main__.py contents from pex.
# Most of this is being robust to python2 / 3.
if hash python3 2>/dev/null; then
    INTERPRETER="python3"
elif hash python 2>/dev/null; then
    INTERPRETER="python"
elif hash pypy 2>/dev/null; then
    INTERPRETER="pypy"
else
    echo "Can't find a usable Python interpreter"
    exit 1
fi
exec $INTERPRETER -c "from third_party.python.pex import pex_builder; print(pex_builder.BOOTSTRAP_ENVIRONMENT.decode())"
