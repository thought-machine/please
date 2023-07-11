go_library(
    name = "metrics",
    srcs = ["prometheus.go"],
    pgo_file = "//:pgo",
    visibility = ["PUBLIC"],
    deps = [
        "///third_party/go/github.com_prometheus_client_golang//prometheus",
        "///third_party/go/github.com_prometheus_client_golang//prometheus/push",
        "///third_party/go/github.com_prometheus_common//expfmt",
        "//src/cli/logging",
        "//src/core",
    ],
)
