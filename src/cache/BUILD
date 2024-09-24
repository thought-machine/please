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
    ],
)

go_test(
    name = "cache_test",
    srcs = glob(["*_test.go"]),
    data = ["test_data"],
    deps = [
        ":cache",
        "///third_party/go/github.com_stretchr_testify//assert",
        "//src/core",
    ],
)
