go_binary(
    name = "jarcat",
    srcs = ["main.go"],
    visibility = ["PUBLIC"],
    deps = [
        "//src/cli",
        "//third_party/go:logging",
        "//tools/jarcat/ar",
        "//tools/jarcat/tar",
        "//tools/jarcat/unzip",
        "//tools/jarcat/zip",
    ],
)

go_binary(
    name = "jarcat_unzip",
    srcs = ["unzip_main.go"],
    visibility = ["//third_party/go:all"],
)
