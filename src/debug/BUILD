go_library(
    name = "debug",
    srcs = glob(
        ["*.go"],
        exclude = ["*_test.go"],
    ),
    visibility = ["PUBLIC"],
    deps = [
        "//src/core",
        "//third_party/go:logging",
        "//src/exec",
        "//src/process",
    ],
)
