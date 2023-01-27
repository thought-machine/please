go_binary(
    name = "http_cache",
    srcs = ["main.go"],
    visibility = ["PUBLIC"],
    deps = [
        "///third_party/go/gopkg.in_op_go-logging.v1//:go-logging.v1",
        "//src/cli",
        "//tools/http_cache/cache",
    ],
)

sh_cmd(
    name = "run_local",
    srcs = [":http_cache"],
    cmd = "exec $(out_location :http_cache) -p 1771 -d /tmp/please_http_cache",
)
