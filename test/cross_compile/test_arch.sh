#!/bin/sh
# This will succeed if the correct arch has been recorded.
exec grep "test_x86" "${PKG_DIR}/arch.txt"
