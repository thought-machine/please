genrule(
    name = "rule_defs",
    srcs = [
        "process.py",
        "//src/parse/args",
    ],
    outs = ["rule_defs.json"],
    cmd = "$TOOL $SRCS > $OUT",
    tools = [CONFIG.DEFAULT_PYTHON_INTERPRETER],
)

# We embed machine-readable information about the rules into the binary.
# Conceptually we could get the interpreter to read back their docstrings
# but we'd rather avoid the cgo link for that, and this embedding is
# easy enough.
go_bindata(
    name = "data",
    srcs = [
        "misc.json",
        ":config_json",
        ":rule_defs",
    ],
    compress = True,
)

go_library(
    name = "help",
    srcs = [
        "completion.go",
        "help.go",
        ":data",
    ],
    visibility = ["PUBLIC"],
    deps = [
        "//src/cli",
        "//src/utils",
        "//third_party/go:go-flags",
        "//third_party/go:logging",
    ],
)

go_test(
    name = "help_test",
    srcs = ["help_test.go"],
    deps = [
        ":help",
        "//third_party/go:testify",
    ],
)

genrule(
    name = "config_json",
    outs = ["config.json"],
    cmd = "$TOOL > $OUTS",
    tools = ["//src/help/config:config_help"],
)
