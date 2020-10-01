go_library(
    name = "format",
    srcs = ["fmt.go"],
    deps = [
        "//src/core",
        "//src/fs",
        "//src/utils",
        "//third_party/go:buildtools",
    ],
    visibility = ["//src/..."],
)

go_test(
    name = "format_test",
    srcs = ["fmt_test.go"],
    deps = [
        ":fmt",
        "//src/core",
        "//third_party/go:testify",
    ],
    data = ["test_data"],
)
