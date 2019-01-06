go_library(
    name = "langserver",
    srcs = glob(
        ["*.go"],
        exclude = ["*_test.go"],
    ),
    visibility = ["//tools/build_langserver/..."],
    deps = [
        "//src/build",
        "//src/cli",
        "//src/core",
        "//src/fs",
        "//src/parse",
        "//src/parse/asp",
        "//src/parse/rules",
        "//src/plz",
        "//src/query",
        "//third_party/go:buildtools",
        "//third_party/go:jsonrpc2",
        "//third_party/go:logging",
        "//third_party/go:queue",
        "//tools/build_langserver/lsp",
    ],
)

go_test(
    name = "langserver_test",
    srcs = glob(
        [
            "*_test.go",
            "setup_test.go",
        ],
        exclude = ["utils_test.go"],
    ),
    data = ["test_data"],
    deps = [
        ":langserver",
        "//src/parse/rules",
        "//third_party/go:testify",
    ],
)

go_test(
    name = "utils_test",
    srcs = ["utils_test.go"],
    data = ["test_data"],
    deps = [
        ":langserver",
        "//third_party/go:testify",
    ],
)

go_test(
    name = "analyzer_test",
    srcs = [
        "analyzer_test.go",
        "setup_test.go",
    ],
    data = ["test_data"],
    deps = [
        ":langserver",
        "//third_party/go:testify",
        "//tools/build_langserver/lsp",
    ],
)

go_test(
    name = "diagnostics_test",
    srcs = [
        "diagnostics_test.go",
        "setup_test.go",
    ],
    data = ["test_data"],
    deps = [
        ":langserver",
        "//third_party/go:testify",
        "//tools/build_langserver/lsp",
    ],
)
