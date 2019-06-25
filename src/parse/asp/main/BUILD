go_binary(
    name = "asp",
    srcs = ["main.go"],
    deps = [
        "//rules",
        "//src/cli",
        "//src/core",
        "//src/parse/asp",
        "//third_party/go:logging",
        "//third_party/go:spew",
    ],
)

go_binary(
    name = "compiler",
    srcs = ["compiler.go"],
    visibility = ["//rules:all"],
    deps = [
        "//src/cli",
        "//src/core",
        "//src/parse/asp",
        "//third_party/go:logging",
    ],
)
