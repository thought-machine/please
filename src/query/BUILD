go_library(
    name = "query",
    srcs = glob(
        ["*.go"],
        exclude = ["*_test.go"],
    ),
    visibility = ["PUBLIC"],
    deps = [
        "//src/build",
        "//src/cli",
        "//src/core",
        "//src/scm",
        "//src/utils",
        "//third_party/go:logging",
    ],
)

go_test(
    name = "print_test",
    srcs = ["print_test.go"],
    deps = [
        ":query",
        "//src/core",
        "//third_party/go:testify",
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
