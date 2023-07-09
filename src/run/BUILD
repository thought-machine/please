go_library(
    name = "run",
    srcs = ["run_step.go"],
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
    data = ["test_data"],
    deps = [
        ":run",
        "///third_party/go/github.com_stretchr_testify//assert",
        "//src/core",
        "//src/process",
    ],
)
