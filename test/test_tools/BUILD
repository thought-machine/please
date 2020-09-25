sh_binary(
    name = "tool",
    srcs = "tool.sh",
)

build_rule(
    name = "unnamed_test",
    outs = ["UNNAMED"],
    cmd = "echo 'wibble wibble wibble' > $OUT",
    no_test_output = True,
    test = True,
    test_cmd = "$TOOL $TEST",
    test_tools = [":tool"],
)

build_rule(
    name = "named_test",
    outs = ["NAMED"],
    cmd = "echo 'wibble wibble wibble' > $OUT",
    no_test_output = True,
    test = True,
    test_cmd = "$TOOLS_DUMMY $TEST",
    test_tools = {
        "DUMMY": [":tool"],
    },
)
