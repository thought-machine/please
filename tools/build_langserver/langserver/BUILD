go_library(
    name = "langserver",
    srcs = glob(
        ["*.go"],
        exclude = ["*_test.go"],
    ),
    deps = [
        "//src/core",
        "//src/fs",
        "//src/parse",
        "//third_party/go:jsonrpc2",
        "//third_party/go:logging",
        "//tools/build_langserver/lsp",
    ],
    visibility = ["//tools/build_langserver/..."],
)

go_test(
    name = "langserver_test",
    srcs = ["handler_test.go"],
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
