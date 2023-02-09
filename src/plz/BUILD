go_library(
    name = "plz",
    srcs = ["plz.go"],
    visibility = ["PUBLIC"],
    deps = [
        "//src/build",
        "//src/cli",
        "//src/cli/logging",
        "//src/core",
        "//src/fs",
        "//src/metrics",
        "//src/parse",
        "//src/remote",
        "//src/test",
        "//third_party/go:cli-init",
    ],
)

go_test(
    name = "plz_test",
    srcs = ["plz_test.go"],
    deps = [
        ":plz",
        "//src/core",
        "//src/cli",
        "//third_party/go:testify",
    ]
)