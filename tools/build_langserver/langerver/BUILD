go_library(
    name = "langserver",
    srcs = glob(
        ["*.go"],
        exclude = ["*_test.go"],
    ),
    deps = [
        "//src:parse",
        "//third_party/go:jsonrpc2",
    ],
)

go_test(
    name = "langserver_test",
    srcs = glob(["*_test.go"]),
    deps = [
        "//third_party/go:testify",
    ],
)
