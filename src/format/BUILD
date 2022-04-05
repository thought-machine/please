go_library(
    name = "format",
    srcs = ["fmt.go"],
    visibility = ["//src/..."],
    deps = [
        "//src/cli/logging",
        "//src/core",
        "//src/fs",
        "//src/plz",
        "//third_party/go:buildtools",
        "//third_party/go:errgroup",
    ],
)

go_test(
    name = "format_test",
    srcs = ["fmt_test.go"],
    data = ["test_data"],
    deps = [
        ":format",
        "//src/core",
        "//third_party/go:testify",
    ],
)
