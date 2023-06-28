go_library(
    name = "format",
    srcs = ["fmt.go"],
    visibility = ["//src/..."],
    deps = [
        "///third_party/go/github.com_bazelbuild_buildtools//build",
        "///third_party/go/golang.org_x_sync//errgroup",
        "//src/cli/logging",
        "//src/core",
        "//src/fs",
        "//src/plz",
    ],
)

go_test(
    name = "format_test",
    srcs = ["fmt_test.go"],
    data = ["test_data"],
    deps = [
        ":format",
        "///third_party/go/github.com_stretchr_testify//assert",
        "///third_party/go/github.com_stretchr_testify//require",
        "//src/core",
    ],
)
