go_library(
    name = "cli",
    srcs = glob(
        ["*.go"],
        exclude = ["*_test.go"],
    ),
    visibility = ["PUBLIC"],
    deps = [
        "//third_party/go:cli-init",
        "//third_party/go:go-flags",
        "//third_party/go:humanize",
        "//third_party/go:logging",
        "//third_party/go:promptui",
        "//third_party/go:semver",
        "//third_party/go:xcrypto",
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

go_test(
    name = "stdin_test",
    srcs = ["stdin_test.go"],
    data = ["test_data"],
    deps = [
        ":cli",
        "//third_party/go:testify",
    ],
)
