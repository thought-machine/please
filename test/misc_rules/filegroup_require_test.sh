#!/usr/bin/env bash
if [ -f test/misc_rules/a.txt ]; then
    exit 0
fi
exit 1
