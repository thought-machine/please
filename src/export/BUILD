go_library(
    name = "export",
    srcs = ["export.go"],
    pgo_file = "//:pgo",
    visibility = ["PUBLIC"],
    deps = [
        "///third_party/go/github.com_please-build_buildtools//build",
        "//src/cli/logging",
        "//src/core",
        "//src/fs",
        "//src/parse",
    ],
)

go_test(
    name = "export_test",
    srcs = ["export_test.go"],
    deps = [
        ":export",
        "///third_party/go/github.com_stretchr_testify//assert",
        "//src/core",
    ],
)
