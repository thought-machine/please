go_library(
    name = "lint",
    srcs = glob(
        ["*.go"],
        exclude = ["*_test.go"],
    ),
    visibility = ["//src/..."],
    deps = [
        "//src/core",
        "//src/fs",
        "//src/process",
        "//third_party/go:deferred_regex",
        "//third_party/go:logging",
    ],
)
