go_library(
    name = "run",
    srcs = [
        "env_other.go",
        "env_windows.go",
        "run_step.go",
    ],
    pgo_file = "//:pgo",
    visibility = ["PUBLIC"],
    deps = [
        "///third_party/go/golang.org_x_sync//errgroup",
        "//src/cli",
        "//src/cli/logging",
        "//src/core",
        "//src/fs",
        "//src/output",
        "//src/process",
    ],
)

go_test(
    name = "run_test",
    srcs = ["run_test.go"],
    data = [":run_test_data"],
    # So //test/windows can run this same binary, cross-built, under Wine.
    visibility = ["//test/windows:all"],
    deps = [
        ":run",
        "///third_party/go/github.com_stretchr_testify//assert",
        "///third_party/go/github.com_stretchr_testify//require",
        "//src/core",
        "//src/process",
    ],
)

# Exposed so //test/windows can put it beside the test binary when running it under Wine.
filegroup(
    name = "run_test_data",
    srcs = ["test_data"],
    test_only = True,
    visibility = ["//test/windows:all"],
)
