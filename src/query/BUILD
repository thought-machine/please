go_library(
    name = "query",
    srcs = glob(
        ["*.go"],
        exclude = ["*_test.go"],
    ),
    pgo_file = "//:pgo",
    visibility = ["PUBLIC"],
    deps = [
        "///third_party/go/github.com_please-build_gcfg//:gcfg",
        "///third_party/go/golang.org_x_exp//maps",
        "//src/build",
        "//src/cli/logging",
        "//src/core",
        "//src/fs",
        "//src/parse",
    ],
)

go_test(
    name = "query_test",
    srcs = glob(["*_test.go"]),
    data = ["completions_test_repo"],
    deps = [
        ":query",
        "///third_party/go/github.com_stretchr_testify//assert",
        "///third_party/go/github.com_stretchr_testify//require",
        "//src/cli",
        "//src/core",
        "//src/parse",
    ],
)
