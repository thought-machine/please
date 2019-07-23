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
        "//src/cache/proto:rpc_cache",
        "//src/cache/tools",
        "//src/cli",
        "//src/core",
        "//src/fs",
        "//src/remote",
        "//third_party/go:atime",
        "//third_party/go:grpc",
        "//third_party/go:humanize",
        "//third_party/go:logging",
    ],
)

go_test(
    name = "rpc_cache_test",
    srcs = ["rpc_cache_test.go"],
    data = [":test_data"],
    labels = ["no_cirrus"],
    deps = [
        ":cache",
        "//third_party/go:grpc",
        "//third_party/go:logging",
        "//third_party/go:testify",
        "//tools/cache/server",
    ],
)

filegroup(
    name = "test_data",
    srcs = ["test_data"],
    test_only = True,
    visibility = ["//tools/cache/..."],
)

go_test(
    name = "http_cache_test",
    srcs = ["http_cache_test.go"],
    data = [":test_data"],
    deps = [
        ":cache",
        "//third_party/go:logging",
        "//tools/cache/server",
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
