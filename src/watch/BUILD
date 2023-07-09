go_library(
    name = "watch",
    srcs = ["watch.go"],
    pgo_file = "//:pgo",
    visibility = ["PUBLIC"],
    deps = [
        "///third_party/go/github.com_fsnotify_fsnotify//:fsnotify",
        "//src/cli",
        "//src/cli/logging",
        "//src/core",
        "//src/fs",
        "//src/process",
        "//src/run",
    ],
)
