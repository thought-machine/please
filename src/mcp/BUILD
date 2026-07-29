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
