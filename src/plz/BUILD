go_library(
    name = "plz",
    srcs = glob(
        ["*.go"],
        exclude = ["*_test.go"],
    ),
    visibility = ["PUBLIC"],
    deps = [
        "//src/build",
        "//src/cli",
        "//src/core",
        "//src/follow",
        "//src/fs",
        "//src/metrics",
        "//src/output",
        "//src/parse",
        "//src/test",
        "//src/utils",
        "//third_party/go:logging",
    ],
)
