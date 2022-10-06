go_library(
    name = "exec",
    srcs = glob(
        ["*.go"],
        exclude = ["*_test.go"],
    ),
    visibility = ["PUBLIC"],
    deps = [
        "//src/cli/logging",
        "//src/core",
        "//src/process",
        "//third_party/go:errgroup",
    ],
)

go_test(
    name = "exec_test",
    srcs = ["exec_test.go"],
    deps = [
        ":exec",
        "//src/build",
        "//src/core",
        "//src/fs",
        "//src/process",
        "//third_party/go:testify",
    ],
)
