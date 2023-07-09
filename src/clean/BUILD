go_library(
    name = "clean",
    srcs = ["clean.go"],
    pgo_file = "//:pgo",
    visibility = ["PUBLIC"],
    deps = [
        "//src/build",
        "//src/cli/logging",
        "//src/core",
        "//src/fs",
        "//src/test",
    ],
)

go_test(
    name = "clean_test",
    srcs = ["clean_test.go"],
    deps = [
        ":clean",
        "///third_party/go/github.com_stretchr_testify//assert",
        "//src/fs",
    ],
)
