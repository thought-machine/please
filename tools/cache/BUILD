go_binary(
    name = "http_cache_server",
    srcs = ["http_server_main.go"],
    visibility = ["PUBLIC"],
    deps = [
        "//src/cli",
        "//third_party/go:logging",
        "//tools/cache/server",
    ],
)

go_binary(
    name = "rpc_cache_server",
    srcs = ["rpc_server_main.go"],
    visibility = ["PUBLIC"],
    deps = [
        "//src/cli",
        "//third_party/go:grpc",
        "//third_party/go:grpc-prometheus",
        "//third_party/go:logging",
        "//third_party/go:prometheus",
        "//tools/cache/cluster",
        "//tools/cache/server",
    ],
)
