go_library(
    name = "debug",
    srcs = glob(
        ["*.go"],
        exclude = ["*_test.go"],
    ),
    visibility = ["PUBLIC"],
    deps = [
        "//src/cli/logging",
        "//src/core",
        "//src/exec",
        "//src/process",
    ],
)
