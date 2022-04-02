go_library(
    name = "cli",
    srcs = glob(
        ["*.go"],
        exclude = ["*_test.go"],
    ),
    visibility = ["PUBLIC"],
    deps = [
        "//src/cli/logging",
        "//third_party/go:cli-init",
        "//third_party/go:deferred_regex",
        "//third_party/go:go-flags",
        "//third_party/go:humanize",
        "//third_party/go:levenshtein",
        "//third_party/go:logging",
        "//third_party/go:promptui",
        "//third_party/go:semver",
        "//third_party/go:xterm",
    ],
)

go_test(
    name = "flags_test",
    srcs = ["flags_test.go"],
    deps = [
        ":cli",
        "//third_party/go:testify",
    ],
)

go_test(
    name = "logging_test",
    srcs = ["logging_test.go"],
    deps = [
        ":cli",
        "//third_party/go:go-flags",
        "//third_party/go:testify",
    ],
)
