go_binary(
    name = "build_langserver",
    srcs = ["langserver_main.go"],
    visibility = ["PUBLIC"],
    deps = [
        "///third_party/go/github.com_sourcegraph_jsonrpc2//:jsonrpc2",
        "///third_party/go/gopkg.in_op_go-logging.v1//:go-logging.v1",
        "//src/cli",
        "//tools/build_langserver/lsp",
    ],
)
