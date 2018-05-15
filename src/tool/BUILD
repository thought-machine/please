go_library(
    name = "tool",
    srcs = ["tool.go"],
    visibility = ["PUBLIC"],
    deps = [
        "//src/core",
        "//third_party/go:go-flags",
        "//third_party/go:logging",
    ],
)

go_test(
    name = "tool_test",
    srcs = ["tool_test.go"],
    deps = [
        ":tool",
        "//src/core",
        "//third_party/go:testify",
    ],
)
