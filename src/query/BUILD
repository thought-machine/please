go_library(
    name = "query",
    srcs = glob(
        ["*.go"],
        exclude = ["*_test.go"],
    ),
    pgo_file = "//:pgo",
    visibility = ["PUBLIC"],
    deps = [
        "///third_party/go/golang.org_x_exp//maps",
        "///third_party/go/github.com_please-build_gcfg//:gcfg",
        "//src/build",
        "//src/cli/logging",
        "//src/core",
        "//src/parse",
        "//src/fs",
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
        "//src/core",
        "//src/parse",
    ],
)
