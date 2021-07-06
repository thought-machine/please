go_library(
    name = "exec",
    srcs = glob(
        ["*.go"],
        exclude = ["*_test.go"],
    ),
    visibility = ["PUBLIC"],
    deps = [
        "//src/core",
        "//src/fs",
        "//src/process",
        "//third_party/go:logging",
    ],
)
