go_library(
    name = "lsp",
    srcs = glob(
        ["*.go"],
        exclude = ["*_test.go"],
    ),
    visibility = ["PUBLIC"],
    deps = [
        "//third_party/go:jsonrpc2",
        "//third_party/go:logging",
    ],
)
