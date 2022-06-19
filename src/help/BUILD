go_library(
    name = "help",
    srcs = glob(
        ["*.go"],
        exclude = ["*_test.go"],
    ),
    visibility = ["PUBLIC"],
    deps = [
        "//rules",
        "//src/cli",
        "//src/cli/logging",
        "//src/core",
        "//src/parse/asp",
        "//src/plz",
        "//third_party/go:deferred_regex",
        "//third_party/go:go-flags",
    ],
)

go_test(
    name = "help_test",
    srcs = [
        "help_test.go",
        "rules_test.go",
    ],
    deps = [
        ":help",
        "//third_party/go:testify",
    ],
)
