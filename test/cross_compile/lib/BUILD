# This contains an arbitrary set of rules intended to exercise
# aspects of the cross-compilation code.

subinclude("//test/build_defs:cc_embed_binary", "///cc//build_defs:c")

go_binary(
    name = "generator",
    srcs = ["main.go"],
)

genrule(
    name = "generated",
    outs = ["gen.txt"],
    cmd = '"$TOOL" > "$OUT"',
    tools = [":generator"],
)

c_embed_binary(
    name = "embed",
    src = ":generated",
)

c_library(
    name = "lib",
    srcs = ["lib.c"],
    hdrs = ["lib.h"],
    visibility = ["//test/cross_compile:all"],
    deps = [":embed"],
)
