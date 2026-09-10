go_library(
    name = "clean",
    srcs = [
        "clean.go",
        "detach_other.go",
        "detach_windows.go",
    ],
    pgo_file = "//:pgo",
    visibility = ["PUBLIC"],
    deps = [
        "///third_party/go/golang.org_x_sys//windows",
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
