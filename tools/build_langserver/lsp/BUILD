go_library(
    name = "lsp",
    srcs = [
        "completion.go",
        "definition.go",
        "diagnostics.go",
        "lsp.go",
        "symbols.go",
        "text.go",
    ],
    visibility = ["//tools/build_langserver/..."],
    deps = [
        "///third_party/go/github.com_please-build_buildtools//build",
        "///third_party/go/github.com_sourcegraph_go-lsp//:go-lsp",
        "///third_party/go/github.com_sourcegraph_jsonrpc2//:jsonrpc2",
        "///third_party/go/gopkg.in_op_go-logging.v1//:go-logging.v1",
        "//rules",
        "//src/core",
        "//src/fs",
        "//src/parse/asp",
        "//src/plz",
        "//tools/build_langserver/lsp/astutils",
    ],
)

go_test(
    name = "lsp_test",
    size = "medium",
    srcs = [
        "definition_test.go",
        "lsp_test.go",
        "symbols_test.go",
    ],
    data = ["test_data"],
    deps = [
        ":lsp",
        "///third_party/go/github.com_sourcegraph_go-lsp//:go-lsp",
        "///third_party/go/github.com_sourcegraph_jsonrpc2//:jsonrpc2",
        "///third_party/go/github.com_stretchr_testify//assert",
        "//src/cli",
        "//src/core",
        "///third_party/go/github.com_please-build_buildtools//build",
    ],
)
