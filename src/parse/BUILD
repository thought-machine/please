go_library(
    name = "parse",
    srcs = [
        "init.go",
        "internal_package.go",
        "parse_step.go",
    ],
    pgo_file = "//:pgo",
    resources = glob(["internal.tmpl"]),
    visibility = ["PUBLIC"],
    deps = [
        "//rules",
        "//rules/bazel",
        "//src/cli",
        "//src/cli/logging",
        "//src/core",
        "//src/fs",
        "//src/parse/asp",
        "//src/version",
    ],
)

go_test(
    name = "parse_step_test",
    srcs = ["parse_step_test.go"],
    resources = ["internal.tmpl"],
    deps = [
        ":parse",
        "///third_party/go/github.com_stretchr_testify//assert",
        "//src/core",
    ],
)
