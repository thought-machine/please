go_library(
    name = "exec",
    srcs = glob(
        ["*.go"],
        exclude = ["*_test.go"],
    ),
    visibility = ["PUBLIC"],
    deps = [
        "//src/core",
        "//src/process",
        "//third_party/go:logging",
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
