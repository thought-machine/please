go_library(
    name = "unzip",
    srcs = ["unzip.go"],
    visibility = ["//tools/jarcat:all"],
    deps = [
        "//third_party/go:xz",
        "//third_party/go/zip",
    ],
)

go_test(
    name = "unzip_test",
    srcs = ["unzip_test.go"],
    data = ["test_data"],
    deps = [
        ":unzip",
        "//third_party/go:testify",
    ],
)
