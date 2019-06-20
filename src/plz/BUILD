go_library(
    name = "plz",
    srcs = [
        "limit.go",
        "plz.go",
    ],
    visibility = ["PUBLIC"],
    deps = [
        "//src/build",
        "//src/cli",
        "//src/core",
        "//src/follow",
        "//src/fs",
        "//src/parse",
        "//src/test",
        "//src/utils",
        "//third_party/go:logging",
    ],
)

go_test(
    name = "limit_test",
    srcs = ["limit_test.go"],
    data = ["test_data"],
    deps = [
        ":plz",
        "//src/core",
        "//third_party/go:testify",
    ],
)
