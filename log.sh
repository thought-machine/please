#!/usr/bin/env bash

set -eu

function notice {
    >&2 echo -e "\033[32m`date +%H:%M:%S.%3N` $1\033[0m"
}
function warn {
    >&2 echo -e "\033[33m`date +%H:%M:%S.%3N` $1\033[0m"
}
