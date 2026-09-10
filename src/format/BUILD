go_library(
    name = "format",
    srcs = ["fmt.go"],
    pgo_file = "//:pgo",
    visibility = ["//src/..."],
    deps = [
        "///third_party/go/github.com_please-build_buildtools//build",
        "///third_party/go/golang.org_x_sync//errgroup",
        "//src/cli/logging",
        "//src/core",
        "//src/fs",
        "//src/plz",
    ],
)

go_test(
    name = "format_test",
    srcs = ["fmt_test.go"],
    data = [":format_test_data"],
    # So //test/windows can run this same binary, cross-built, under Wine.
    visibility = ["//test/windows:all"],
    deps = [
        ":format",
        "///third_party/go/github.com_stretchr_testify//assert",
        "///third_party/go/github.com_stretchr_testify//require",
        "//src/core",
    ],
)

# Exposed so //test/windows can put it beside the test binary when running it under Wine.
filegroup(
    name = "format_test_data",
    srcs = ["test_data"],
    test_only = True,
    visibility = ["//test/windows:all"],
)
