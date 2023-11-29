go_library(
    name = "build",
    srcs = [
        "build_step.go",
        "filegroup.go",
        "incrementality.go",
    ],
    pgo_file = "//:pgo",
    visibility = ["PUBLIC"],
    deps = [
        "///third_party/go/github.com_hashicorp_go-multierror//:go-multierror",
        "///third_party/go/github.com_hashicorp_go-retryablehttp//:go-retryablehttp",
        "//src/cli",
        "//src/cli/logging",
        "//src/core",
        "//src/fs",
        "//src/generate",
        "//src/metrics",
        "//src/process",
    ],
)

go_test(
    name = "build_test",
    srcs = [
        "build_step_test.go",
        "incrementality_test.go",
        "remote_file_test.go",
    ],
    data = ["test_data"],
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
