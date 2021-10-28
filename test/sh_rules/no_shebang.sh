true

# This checks that the top line of the script isn't reused for building the target unless it's a shebang.
if ! grep -ax "true" $0 | wc -l | grep -x "1" >/dev/null; then
    echo "The top line of the file is being reused and it's not a shebang" >&2
    exit 1
fi

