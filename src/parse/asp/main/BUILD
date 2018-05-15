go_binary(
    name = "asp",
    srcs = ["main.go"],
    deps = [
        "//src/cli",
        "//src/core",
        "//src/parse/asp",
        "//src/parse/rules",
        "//third_party/go:logging",
        "//third_party/go:spew",
    ],
)

go_binary(
    name = "compiler",
    srcs = ["compiler.go"],
    visibility = ["//src/parse/rules:all"],
    deps = [
        "//src/cli",
        "//src/core",
        "//src/parse/asp",
        "//third_party/go:logging",
    ],
)
