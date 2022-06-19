go_library(
    name = "cache",
    srcs = glob(
        ["*.go"],
        exclude = [
            "*_test.go",
        ],
    ),
    visibility = ["PUBLIC"],
    deps = [
        "//src/clean",
        "//src/cli",
        "//src/cli/logging",
        "//src/core",
        "//src/fs",
        "//src/process",
        "//third_party/go:atime",
        "//third_party/go:go-retryablehttp",
        "//third_party/go:humanize",
    ],
)

filegroup(
    name = "test_data",
    srcs = ["test_data"],
    test_only = True,
)

go_test(
    name = "http_cache_test",
    srcs = ["http_cache_test.go"],
    data = [":test_data"],
    deps = [
        ":cache",
        "//src/cli/logging",
        "//third_party/go:testify",
    ],
)

go_test(
    name = "async_cache_test",
    srcs = ["async_cache_test.go"],
    deps = [
        ":cache",
        "//third_party/go:testify",
    ],
)

go_test(
    name = "dir_cache_test",
    srcs = ["dir_cache_test.go"],
    deps = [
        ":cache",
        "//third_party/go:testify",
    ],
)

go_test(
    name = "cmd_cache_test",
    srcs = ["cmd_cache_test.go"],
    data = [":test_data"],
    deps = [
        ":cache",
        "//third_party/go:testify",
    ],
)
