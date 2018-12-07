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
        "//src/hashes",
        "//src/metrics",
        "//src/output",
        "//src/parse",
        "//src/run",
        "//src/test",
        "//src/utils",
        "//third_party/go:logging",
    ],
)
