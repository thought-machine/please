go_binary(
    name = "http_cache",
    srcs = ["main.go"],
    visibility = ["PUBLIC"],
    deps = [
        "//src/cli",
        "//src/cli/logging",
        "//tools/http_cache/cache",
    ],
)

sh_cmd(
    name = "run_local",
    data = [":http_cache"],
    cmd = r"exec \\$DATA -p 1771 -d /tmp/please_http_cache",
)
