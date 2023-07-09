go_library(
    name = "cache",
    srcs = [
        "async_cache.go",
        "cache.go",
        "cmd_cache.go",
        "dir_cache.go",
        "http_cache.go",
        "noop.go",
    ],
    pgo_file = "//:pgo",
    visibility = ["PUBLIC"],
    deps = [
        "///third_party/go/github.com_djherbis_atime//:atime",
        "///third_party/go/github.com_dustin_go-humanize//:go-humanize",
        "///third_party/go/github.com_hashicorp_go-retryablehttp//:go-retryablehttp",
        "//src/clean",
        "//src/cli",
        "//src/cli/logging",
        "//src/core",
        "//src/fs",
        "//src/process",
    ],
)

filegroup(
    name = "test_data",
    srcs = ["test_data"],
    test_only = True,
)

go_test(
    name = "cache_test",
    srcs = [
        "async_cache_test.go",
        "cmd_cache_test.go",
        "dir_cache_test.go",
        "http_cache_test.go",
    ],
    data = [":test_data"],
    deps = [
        ":cache",
        "///third_party/go/github.com_stretchr_testify//assert",
        "//src/core",
    ],
)
