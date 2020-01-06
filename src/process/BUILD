go_library(
    name = "process",
    srcs = glob(
        ["*.go"],
        exclude = ["*_test.go"],
    ),
    visibility = ["PUBLIC"],
    deps = [
        "//src/cli",
        "//third_party/go:logging",
    ],
)

go_test(
    name = "process_test",
    srcs = ["process_test.go"],
    deps = [
        ":process",
        "//third_party/go:testify",
    ],
)

go_test(
    name = "progress_test",
    srcs = ["progress_test.go"],
    deps = [
        ":process",
        "//third_party/go:testify",
    ],
)
