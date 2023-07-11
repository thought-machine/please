go_library(
    name = "tool",
    srcs = ["tool.go"],
    pgo_file = "//:pgo",
    visibility = ["PUBLIC"],
    deps = [
        "///third_party/go/github.com_thought-machine_go-flags//:go-flags",
        "//src/cli/logging",
        "//src/core",
        "//src/fs",
    ],
)

go_test(
    name = "tool_test",
    srcs = ["tool_test.go"],
    deps = [
        ":tool",
        "///third_party/go/github.com_stretchr_testify//assert",
        "//src/core",
    ],
)
