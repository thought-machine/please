subinclude('//build_defs:go_bindata')

genrule(
    name = 'rule_defs',
    srcs = [
        'process.py',
        '//src/parse:rule_args',
    ],
    outs = ['rule_defs.json'],
    cmd = 'python $SRCS > $OUT',
)

# We embed machine-readable information about the rules into the binary.
# Conceptually we could get the interpreter to read back their docstrings
# but we'd rather avoid the cgo link for that, and this embedding is
# easy enough.
go_bindata(
    name = 'data',
    srcs = [
        'misc.json',
        ':config_json',
        ':help_json',
        ':rule_defs',
    ],
    compress = True,
)

go_library(
    name = 'help',
    srcs = [
        'completion.go',
        'help.go',
        ':data',
    ],
    deps = [
        '//src/cli',
        '//src/utils',
        '//third_party/go:go-flags',
        '//third_party/go:logging',
    ],
    visibility = ['PUBLIC'],
)

go_test(
    name = 'help_test',
    srcs = ['help_test.go'],
    deps = [
        ':help',
        '//third_party/go:testify',
    ],
)

genrule(
    name = 'config_json',
    outs = ['config.json'],
    cmd = '$TOOL > $OUTS',
    tools = ['//src/help/config:config_help'],
)

filegroup(
    name = 'help_json',
    srcs = ['//tools/linter:help_topics'],
)
