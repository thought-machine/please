go_library(
    name = "build",
    srcs = glob(
        ["*.go"],
        exclude = ["*_test.go"],
    ),
    visibility = ["PUBLIC"],
    deps = [
        "//src/cli/logging",
        "//src/cli",
        "//src/core",
        "//src/fs",
        "//src/generate",
        "//src/metrics",
        "//src/process",
        "//src/worker",
        "//third_party/go:go-multierror",
        "//third_party/go:go-retryablehttp",
        "//third_party/go:protobuf",
        "//third_party/go:shlex",
    ],
)

go_test(
    name = "incrementality_test",
    srcs = ["incrementality_test.go"],
    deps = [
        ":build",
    ],
)

go_test(
    name = "build_step_test",
    srcs = [
        "build_step_test.go",
        "remote_file_test.go",
    ],
    data = ["test_data"],
    deps = [
        ":build",
        "//src/core",
        "//src/fs",
        "//third_party/go:logging",
        "//third_party/go:testify",
    ],
)

go_test(
    name = "build_step_stress_test",
    srcs = ["build_step_stress_test.go"],
    external = True,
    deps = [
        ":build",
        "//src/cli/logging",
        "//src/core",
        "//src/output",
        "//src/plz",
        "//third_party/go:testify",
    ],
)
