subinclude('//build_defs:go_bindata')

# We embed machine-readable information about the rules into the binary.
# Conceptually we could get the interpreter to read back their docstrings
# but we'd rather avoid the cgo link for that, and this embedding is
# easy enough.
go_bindata(
    name = 'rule_defs',
    srcs = ['//src/parse:rule_args'],
    compress = True,
    prefix = 'src/parse',
)

go_library(
    name = 'help',
    srcs = ['help.go'],
    deps = [
        ':rule_defs',
    ],
    visibility = ['PUBLIC'],
)
