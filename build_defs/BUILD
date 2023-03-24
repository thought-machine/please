filegroup(
    name = "benchmark",
    srcs = ["benchmark.build_defs"],
    visibility = ["PUBLIC"],
)

genrule(
    name = "version",
    srcs = ["//:version"],
    outs = ["version.build_defs"],
    cmd = "echo \"VERSION = '`cat $SRCS`'\" > \"$OUT\"",
    visibility = ["PUBLIC"],
)
