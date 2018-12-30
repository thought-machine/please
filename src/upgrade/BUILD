go_library(
    name = "upgrade",
    srcs = ["upgrade.go"],
    visibility = ["PUBLIC"],
    deps = [
        "//src/core",
        "//third_party/go:logging",
    ],
)

go_test(
    name = "upgrade_test",
    srcs = ["upgrade_test.go"],
    deps = [
        ":upgrade",
        "//third_party/go:testify",
    ],
)
