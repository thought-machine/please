go_library(
    name = "upgrade",
    srcs = ["upgrade.go"],
    deps = [
        "//third_party/go:logging",
        "//src/core",
    ],
    visibility = ["PUBLIC"],
)

go_test(
    name = "upgrade_test",
    srcs = ["upgrade_test.go"],
    deps = [
        "//third_party/go:testify",
        ":upgrade",
    ],
)
