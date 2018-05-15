go_binary(
    name = "arg_printer",
    srcs = ["main.go"],
    deps = [
        "//src/core",
        "//src/parse",
    ],
)

genrule(
    name = "args",
    outs = ["rule_args.json"],
    cmd = "$TOOL query rules > $OUT",
    tools = [":arg_printer"],
    visibility = ["PUBLIC"],
)
