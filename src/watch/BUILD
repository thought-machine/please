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

go_test(
    name = "watch_test",
    srcs = ["watch_test.go"],
    # So //test/windows can run this same binary, cross-built, under Wine, which is the only
    # platform where the assertion means anything.
    visibility = ["//test/windows:all"],
    deps = [
        ":watch",
        "///third_party/go/github.com_stretchr_testify//assert",
    ],
)
