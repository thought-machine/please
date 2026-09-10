go_library(
    name = "build",
    srcs = [
        "build_step.go",
        "clean_tmpdir_darwin.go",
        "clean_tmpdir_other.go",
        "filegroup.go",
        "incrementality.go",
    ],
    pgo_file = "//:pgo",
    visibility = ["PUBLIC"],
    deps = [
        "///third_party/go/github.com_hashicorp_go-multierror//:go-multierror",
        "///third_party/go/github.com_hashicorp_go-retryablehttp//:go-retryablehttp",
        "//src/audit",
        "//src/cli",
        "//src/cli/logging",
        "//src/core",
        "//src/fs",
        "//src/generate",
        "//src/metrics",
        "//src/process",
        "//src/version",
    ],
)

go_test(
    name = "build_test",
    srcs = [
        "build_step_test.go",
        "incrementality_test.go",
        "remote_file_test.go",
    ],
    data = [":test_data"],
    # So //test/windows can run this same binary, cross-built, under Wine.
    visibility = ["//test/windows:all"],
    deps = [
        ":build",
        "///third_party/go/github.com_stretchr_testify//assert",
        "///third_party/go/github.com_stretchr_testify//require",
        "///third_party/go/gopkg.in_op_go-logging.v1//:go-logging.v1",
        "//src/core",
        "//src/fs",
    ],
)

go_test(
    name = "build_step_stress_test",
    srcs = ["build_step_stress_test.go"],
    external = True,
    deps = [
        "///third_party/go/github.com_stretchr_testify//assert",
        "//src/cli/logging",
        "//src/core",
        "//src/fs",
        "//src/plz",
    ],
)

# Exposed so //test/windows can put it beside the test binary when running it under Wine.
filegroup(
    name = "test_data",
    srcs = ["test_data"],
    test_only = True,
    visibility = ["//test/windows:all"],
)
