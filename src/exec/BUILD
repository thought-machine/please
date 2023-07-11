go_library(
    name = "exec",
    srcs = ["exec.go"],
    pgo_file = "//:pgo",
    visibility = ["PUBLIC"],
    deps = [
        "///third_party/go/golang.org_x_sync//errgroup",
        "//src/cli/logging",
        "//src/core",
        "//src/process",
    ],
)

go_test(
    name = "exec_test",
    srcs = ["exec_test.go"],
    deps = [
        ":exec",
        "///third_party/go/github.com_stretchr_testify//assert",
        "//src/build",
        "//src/core",
        "//src/fs",
        "//src/process",
    ],
)
