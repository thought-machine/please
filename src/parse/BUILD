go_library(
    name = "parse",
    srcs = [
        "init.go",
        "internal_package.go",
        "parse_step.go",
    ],
    pgo_file = "//:pgo",
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
        "//src/version",
    ],
)
