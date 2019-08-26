go_library(
    name = "lsp",
    srcs = glob(
        ["*.go"],
        exclude = ["*_test.go"],
    ),
    visibility = ["//tools/build_langserver/..."],
    deps = [
        "//src/core",
        "//src/help",
        "//src/parse/asp",
        "//src/plz",
        "//rules",
        "//third_party/go:buildtools",
        "//third_party/go:jsonrpc2",
        "//third_party/go:logging",
        "//third_party/go:lsp",
    ],
)

go_test(
    name = "lsp_test",
    srcs = glob(["*_test.go"]),
    data = ["test_data"],
    deps = [
        ":lsp",
        "//src/cli",
        "//third_party/go:jsonrpc2",
        "//third_party/go:lsp",
        "//third_party/go:testify",
    ],
)
