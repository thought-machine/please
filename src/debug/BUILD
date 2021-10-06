go_library(
    name = "debug",
    srcs = glob(
        ["*.go"],
        exclude = ["*_test.go"],
    ),
    visibility = ["PUBLIC"],
    deps = [
        "//src/core",
        "//src/exec",
        "//src/process",
        "//third_party/go:logging",
    ],
)
