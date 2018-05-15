go_library(
    name = "query",
    srcs = glob(
        ["*.go"],
        exclude = ["*_test.go"],
    ),
    visibility = ["PUBLIC"],
    deps = [
        "//src/build",
        "//src/core",
        "//src/utils",
        "//third_party/go:logging",
        "//third_party/go:shlex",
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
