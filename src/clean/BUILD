go_library(
    name = "clean",
    srcs = ["clean.go"],
    visibility = ["PUBLIC"],
    deps = [
        "//src/build",
        "//src/core",
        "//src/fs",
        "//src/test",
        "//third_party/go:logging",
    ],
)

go_test(
    name = "clean_test",
    srcs = ["clean_test.go"],
    deps = [
        ":clean",
        "//src/fs",
        "//third_party/go:testify",
    ],
)
