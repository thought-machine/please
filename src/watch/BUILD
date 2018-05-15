go_library(
    name = "watch",
    srcs = ["watch.go"],
    visibility = ["PUBLIC"],
    deps = [
        "//src/core",
        "//src/fs",
        "//third_party/go:concurrent-map",
        "//third_party/go:fsnotify",
        "//third_party/go:logging",
    ],
)

go_test(
    name = "watch_test",
    srcs = ["watch_test.go"],
    deps = [
        ":watch",
        "//src/core",
        "//third_party/go:testify",
    ],
)
