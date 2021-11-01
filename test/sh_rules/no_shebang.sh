true

# This checks that the top line of the script isn't reused for building the target as it's not a shebang.
if [ "`grep -axc "true" $0`" != "1" ]; then
    echo "The top line of the original file is being reused and it's not a shebang" >&2
    exit 1
fi

