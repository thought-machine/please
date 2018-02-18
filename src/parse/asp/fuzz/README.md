Fuzz testing for asp
--------------------

This uses https://github.com/dvyukov/go-fuzz to fuzz test the
BUILD file lexer / parser / interpreter. We build an initial corpus
from the BUILD files in this repo to give it somewhere to start from.

Run fuzz.sh from the repo root to make it go.
