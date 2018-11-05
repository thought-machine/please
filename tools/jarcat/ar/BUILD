go_library(
    name = "ar",
    srcs = ["ar.go"],
    visibility = ["//tools/jarcat/..."],
    deps = [
        "//src/fs",
        "//third_party/go:ar",
        "//third_party/go:logging",
    ],
)

go_test(
    name = "ar_test",
    srcs = ["ar_test.go"],
    data = ["test_data"],
    deps = [
        ":ar",
        "//third_party/go:testify",
    ],
)
