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
        ':rule_defs',
    ],
    compress = True,
    prefix = '$PKG',
)

go_library(
    name = 'help',
    srcs = [
        'help.go',
        ':data',
    ],
    deps = [
        '//src/utils',
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
