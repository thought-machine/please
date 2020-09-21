go_library(
    name = "utils",
    srcs = glob(
        ["*.go"],
        exclude = ["*_test.go"],
    ),
    visibility = ["PUBLIC"],
    deps = [
        "//src/cli",
        "//src/core",
        "//src/fs",
        "//src/scm",
        "//third_party/go:levenshtein",
        "//third_party/go:logging",
    ],
)
