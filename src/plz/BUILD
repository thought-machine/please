go_library(
    name = "plz",
    srcs = glob(
        ["*.go"],
        exclude = ["*_test.go"],
    ),
    visibility = ["PUBLIC"],
    deps = [
        "//src/build",
        "//src/core",
        "//src/fs",
        "//src/parse",
        "//src/cli",
        "//src/test",
        "//src/follow",
        "//src/metrics",
        "//src/output",
        "//src/utils",
        "//third_party/go:logging",
    ],
)
