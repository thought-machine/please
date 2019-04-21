go_library(
    name = "scm",
    srcs = glob(
        ["*.go"],
        exclude = ["*_test.go"],
    ),
    visibility = ["PUBLIC"],
    deps = [
        "//src/fs",
        "//third_party/go:logging",
    ],
)
