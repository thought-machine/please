go_library(
    name = "scm",
    srcs = glob(
        ["*.go"],
        exclude = ["*_test.go"],
    ),
    visibility = ["PUBLIC"],
    deps = [
        "//src/core",
        "//third_party/go:logging",
    ],
)
