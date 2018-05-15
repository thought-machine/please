go_library(
    name = "zip",
    srcs = ["writer.go"],
    visibility = [
        "//tools/jarcat:all",
        "//tools/please_pex/pex",
    ],
    deps = [
        "//src/fs",
        "//third_party/go:logging",
        "//third_party/go:testify",
        "//third_party/go/zip",
    ],
)

go_test(
    name = "writer_test",
    srcs = ["writer_test.go"],
    data = [
        "test_data",
        "test_data_2",
        "test_data_3",
    ],
    deps = [
        ":zip",
    ],
)
