go_library(
    name = "cache",
    srcs = glob(
        ["*.go"],
        exclude = ["*_test.go"],
    ),
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

go_test(
    name = "cache_test",
    srcs = glob(["*_test.go"]),
    data = [":cache_test_data"],
    # So //test/windows can run this same binary, cross-built, under Wine.
    visibility = ["//test/windows:all"],
    deps = [
        ":cache",
        "///third_party/go/github.com_stretchr_testify//assert",
        "//src/core",
    ],
)

# Exposed so //test/windows can put it beside the test binary when running it under Wine.
filegroup(
    name = "cache_test_data",
    srcs = ["test_data"],
    test_only = True,
    visibility = ["//test/windows:all"],
)
