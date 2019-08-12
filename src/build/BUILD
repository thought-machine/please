go_library(
    name = "build",
    srcs = glob(
        ["*.go"],
        exclude = ["*_test.go"],
    ),
    visibility = ["PUBLIC"],
    deps = [
        "//src/cache",
        "//src/core",
        "//src/fs",
        "//src/remote",
        "//src/worker",
        "//third_party/go:go-multierror",
        "//third_party/go:logging",
        "//third_party/go:protobuf",
        "//third_party/go:shlex",
    ],
)

go_test(
    name = "command_replacements_test",
    srcs = ["command_replacements_test.go"],
    deps = [
        ":build",
        "//third_party/go:testify",
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
    srcs = ["build_step_test.go"],
    data = ["test_data"],
    deps = [
        ":build",
        "//src/core",
        "//src/fs",
        "//src/output",
        "//third_party/go:testify",
    ],
)

go_test(
    name = "build_step_stress_test",
    srcs = ["build_step_stress_test.go"],
    external = True,
    deps = [
        ":build",
        "//src/core",
        "//src/output",
        "//src/plz",
        "//third_party/go:logging",
        "//third_party/go:testify",
    ],
)
