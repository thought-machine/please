go_library(
    name = "fmt",
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
    name = "fmt_test",
    srcs = ["fmt_test.go"],
    deps = [
        ":fmt",
        "//src/core",
        "//third_party/go:testify",
    ],
    data = ["test_data"],
)
