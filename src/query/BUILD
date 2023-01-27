go_library(
    name = "query",
    srcs = [
        "alltargets.go",
        "changes.go",
        "completions.go",
        "config.go",
        "deps.go",
        "filter.go",
        "graph.go",
        "inputs.go",
        "outputs.go",
        "print.go",
        "query_step.go",
        "reverse_deps.go",
        "roots.go",
        "somepath.go",
        "whatinputs.go",
        "whatoutputs.go",
    ],
    visibility = ["PUBLIC"],
    deps = [
        "///third_party/go/github.com_please-build_gcfg//:gcfg",
        "//src/build",
        "//src/cli/logging",
        "//src/core",
        "//src/fs",
    ],
)

go_test(
    name = "print_test",
    srcs = [
        "changes_test.go",
        "completions_test.go",
        "deps_test.go",
        "graph_test.go",
        "print_test.go",
        "reverse_deps_test.go",
        "whatinputs_test.go",
        "whatoutputs_test.go",
    ],
    deps = [
        ":query",
        "///third_party/go/github.com_stretchr_testify//assert",
        "///third_party/go/github.com_stretchr_testify//require",
        "//src/core",
        "//src/parse",
    ],
)

go_test(
    name = "graph_test",
    srcs = ["graph_test.go"],
    deps = [
        ":query",
        "//src/core",
        "//third_party/go:testify",
    ],
)

go_test(
    name = "whatoutputs_test",
    srcs = ["whatoutputs_test.go"],
    deps = [
        ":query",
        "//src/core",
        "//third_party/go:testify",
    ],
)

go_test(
    name = "whatinputs_test",
    srcs = ["whatinputs_test.go"],
    deps = [
        ":query",
        "//src/core",
        "//third_party/go:testify",
    ],
)

go_test(
    name = "changes_test",
    srcs = ["changes_test.go"],
    deps = [
        ":query",
        "//src/core",
        "//third_party/go:testify",
    ],
)

go_test(
    name = "reverse_deps_test",
    srcs = ["reverse_deps_test.go"],
    deps = [
        ":query",
        "//src/core",
        "//third_party/go:testify",
    ],
)

go_test(
    name = "completions_test",
    srcs = ["completions_test.go"],
    data = ["completions_test_repo"],
    deps = [
        ":query",
        "//src/core",
        "//third_party/go:testify",
    ],
)
