go_library(
    name = "export",
    srcs = glob(
        ["*.go"],
        exclude = ["*_test.go"],
    ),
    pgo_file = "//:pgo",
    visibility = ["PUBLIC"],
    deps = [
        "///third_party/go/github.com_please-build_buildtools//build",
        "//src/cli",
        "//src/cli/logging",
        "//src/core",
        "//src/fs",
        "//src/parse",
        "//src/parse/asp",
    ],
)

go_test(
    name = "export_test",
    srcs = ["export_test.go"],
    data = [":export_test_data"],
    # So //test/windows can run this same binary, cross-built, under Wine.
    visibility = ["//test/windows:all"],
    deps = [
        ":export",
        "///third_party/go/github.com_stretchr_testify//assert",
        "//src/core",
        "//src/parse/asp",
    ],
)

# Exposed so //test/windows can put it beside the test binary when running it under Wine.
filegroup(
    name = "export_test_data",
    srcs = ["test_data"],
    test_only = True,
    visibility = ["//test/windows:all"],
)
