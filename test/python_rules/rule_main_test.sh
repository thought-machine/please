OUTPUT="`test/python_rules/rule_main_pex.pex`"
if [ "$OUTPUT" != "hello" ]; then
    echo "Unexpected output: $OUTPUT"
    exit 1
fi
