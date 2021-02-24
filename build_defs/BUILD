filegroup(
    name = "multiversion_wheel",
    srcs = ["multiversion_wheel.build_defs"],
    visibility = ["//third_party/python:all"],
)

genrule(
    name = "version",
    srcs = ["//:version"],
    outs = ["version.build_defs"],
    cmd = "echo \"VERSION = '`cat $SRCS`'\" > \"$OUT\"",
    visibility = ["PUBLIC"],
)

filegroup(
    name = "cc_embed_binary",
    srcs = ["cc_embed_binary.build_defs"],
    visibility = ["PUBLIC"],
)
