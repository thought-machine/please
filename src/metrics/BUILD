go_library(
    name = "metrics",
    srcs = ["prometheus.go"],
    visibility = ["PUBLIC"],
    deps = [
        "//src/core",
        "//third_party/go:logging",
        "//third_party/go:prometheus",
        "//third_party/go:shlex",
    ],
)

go_test(
    name = "prometheus_test",
    srcs = ["prometheus_test.go"],
    flaky = True,
    deps = [
        ":metrics",
        "//third_party/go:testify",
    ],
)
