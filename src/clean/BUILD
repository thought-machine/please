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
        "//src/build",
        "//src/cli/logging",
        "//src/core",
        "//src/fs",
        "//src/test",
    ] + (["///third_party/go/golang.org_x_sys//windows"] if is_platform(os = "windows") else []),
)

go_test(
    name = "clean_test",
    srcs = ["clean_test.go"],
    # So //test/windows can run this same binary, cross-built, under Wine.
    visibility = ["//test/windows:all"],
    deps = [
        ":clean",
        "///third_party/go/github.com_stretchr_testify//assert",
        "//src/fs",
    ],
)
