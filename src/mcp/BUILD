go_library(
    name = "mcp",
    srcs = [
        "mcp.go",
        "tools.go",
    ],
    pgo_file = "//:pgo",
    visibility = ["PUBLIC"],
    deps = [
        "///third_party/go/github.com_modelcontextprotocol_go-sdk//mcp",
        "//src/cli/logging",
        "//src/core",
        "//src/parse",
        "//src/plz",
        "//src/query",
        "//src/version",
    ],
)

go_test(
    name = "mcp_test",
    srcs = ["mcp_test.go"],
    external = True,
    deps = [
        ":mcp",
        "///third_party/go/github.com_modelcontextprotocol_go-sdk//mcp",
        "///third_party/go/github.com_stretchr_testify//assert",
        "///third_party/go/github.com_stretchr_testify//require",
        "//src/core",
    ],
)
