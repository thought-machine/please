go_library(
    name = "parse",
    srcs = [
        "init.go",
        "internal_package.go",
        "parse_step.go",
        "suggest.go",
    ],
    visibility = ["PUBLIC"],
    deps = [
        "//rules",
        "//rules/bazel",
        "//src/cli",
        "//src/core",
        "//src/fs",
        "//src/parse/asp",
        "//src/utils",
        "//src/worker",
        "//third_party/go:logging",
    ],
)

go_test(
    name = "parse_step_test",
    srcs = ["parse_step_test.go"],
    deps = [
        ":parse",
        "//src/core",
        "//third_party/go:testify",
    ],
)

go_test(
    name = "suggest_test",
    srcs = ["suggest_test.go"],
    deps = [
        ":parse",
        "//third_party/go:testify",
    ],
)
