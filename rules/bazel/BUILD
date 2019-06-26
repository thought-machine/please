go_bindata(
    name = "builtin_data",
    srcs = glob(["*.build_defs"]),
)

go_library(
    name = "bazel",
    srcs = [":builtin_data"],
    visibility = [
        "//src/parse/...",
    ],
)
