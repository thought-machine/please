# This contains an arbitrary set of rules intended to exercise
# aspects of the cross-compilation code.

go_binary(
    name = 'generator',
    srcs = ['main.go'],
)

genrule(
    name = 'generated',
    outs = ['gen.txt'],
    cmd = '$TOOL > $OUT',
    tools = [':generator'],
)

c_embed_binary(
    name = 'embed',
    src = ':generated',
)

c_library(
    name = 'lib',
    srcs = ['lib.c'],
    hdrs = ['lib.h'],
    deps = [':embed'],
    visibility = ['//test/cross_compile:all'],
)
