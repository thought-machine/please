go_library(
    name = "parse",
    srcs = [
        "init.go",
        "internal_package.go",
        "parse_step.go",
        "suggest.go",
    ],
    resources = ["internal.tmpl"],
    visibility = ["PUBLIC"],
    deps = [
        "//rules",
        "//rules/bazel",
        "//src/cli",
        "//src/cli/logging",
        "//src/core",
        "//src/fs",
        "//src/parse/asp",
        "//src/worker",
        "//third_party/go:semver",
    ],
)

go_test(
    name = "parse_step_test",
    srcs = ["parse_step_test.go"],
    resources = ["internal.tmpl"],
    deps = [
        ":parse",
        "//src/core",
        "//third_party/go:testify",
    ],
)

go_test(
    name = "suggest_test",
    srcs = ["suggest_test.go"],
    resources = ["internal.tmpl"],
    deps = [
        ":parse",
        "//third_party/go:testify",
    ],
)
