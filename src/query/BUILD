go_library(
    name = "query",
    srcs = glob(
        ["*.go"],
        exclude = ["*_test.go"],
    ),
    pgo_file = "//:pgo",
    visibility = ["PUBLIC"],
    deps = [
        "///third_party/go/github.com_please-build_gcfg//:gcfg",
        "///third_party/go/golang.org_x_exp//maps",
        "//src/build",
        "//src/cli",
        "//src/cli/logging",
        "//src/core",
        "//src/fs",
        "//src/parse",
    ],
)

go_test(
    name = "query_test",
    srcs = glob(["*_test.go"]),
    data = [":query_test_data"],
    # So //test/windows can run this same binary, cross-built, under Wine.
    visibility = ["//test/windows:all"],
    deps = [
        ":query",
        "///third_party/go/github.com_stretchr_testify//assert",
        "///third_party/go/github.com_stretchr_testify//require",
        "//src/cli",
        "//src/core",
        "//src/parse",
    ],
)

# Exposed so //test/windows can put it beside the test binary when running it under Wine.
filegroup(
    name = "query_test_data",
    srcs = ["completions_test_repo"],
    test_only = True,
    visibility = ["//test/windows:all"],
)
