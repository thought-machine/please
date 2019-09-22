go_binary(
    name = "build_langserver",
    srcs = ["langserver_main.go"],
    visibility = ["PUBLIC"],
    deps = [
        "//src/cli",
        "//third_party/go:jsonrpc2",
        "//third_party/go:logging",
        "//tools/build_langserver/lsp",
    ],
)
