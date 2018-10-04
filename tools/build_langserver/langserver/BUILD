go_library(
    name = "langserver",
    srcs = glob(
        ["*.go"],
        exclude = ["*_test.go"],
    ),
    visibility = ["//tools/build_langserver/..."],
    deps = [
        "//src/core",
        "//src/fs",
        "//src/parse",
        "//src/parse/asp",
        "//third_party/go:jsonrpc2",
        "//third_party/go:logging",
        "//tools/build_langserver/lsp",
    ],
)

go_test(
    name = "langserver_test",
    srcs = ["handler_test.go",
            "analyzer_test.go"],
    deps = [
        ":langserver",
        "//third_party/go:testify",
    ],
)

go_test(
    name = "utils_test",
    srcs = ["utils_test.go"],
    deps = [
        ":langserver",
        "//third_party/go:testify",
    ],
)
