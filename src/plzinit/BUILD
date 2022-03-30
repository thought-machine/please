go_library(
    name = "plzinit",
    srcs = glob(
        ["*.go"],
        exclude = ["*_test.go"],
    ),
    visibility = ["PUBLIC"],
    deps = [
        "//src/assets",
        "//src/cli",
        "//src/cli/logging",
        "//src/core",
        "//src/fs",
        "//src/plz",
        "//src/scm",
        "//third_party/go:gcfg",
        "//third_party/go:levenshtein",
    ],
)

go_test(
    name = "plzinit_test",
    srcs = ["init_test.go"],
    deps = [
        ":plzinit",
        "//src/fs",
        "//third_party/go:testify",
    ],
)
