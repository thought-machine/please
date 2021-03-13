go_library(
    name = "output",
    srcs = glob(
        ["*.go"],
        exclude = ["*_test.go"],
    ),
    visibility = ["PUBLIC"],
    deps = [
        "//src/cli",
        "//src/core",
        "//src/test",
        "//third_party/go:go-flags",
        "//third_party/go:humanize",
        "//third_party/go:logging",
        "//third_party/go:xcrypto",
    ],
)

go_test(
    name = "interactive_display_test",
    srcs = ["interactive_display_test.go"],
    deps = [
        ":output",
        "//third_party/go:testify",
    ],
)

go_test(
    name = "shell_output_test",
    srcs = ["shell_output_test.go"],
    deps = [
        ":output",
        "//src/core",
        "//third_party/go:testify",
    ],
)
