go_library(
    name = "plz",
    srcs = glob(
        ["*.go"],
        exclude = ["*_test.go"],
    ),
    pgo_file = "//:pgo",
    visibility = ["PUBLIC"],
    deps = [
        "///third_party/go/github.com_peterebden_go-cli-init_v5//flags",
        "///third_party/go/golang.org_x_sync//errgroup",
        "//src/build",
        "//src/cli",
        "//src/cli/logging",
        "//src/cmap",
        "//src/core",
        "//src/fs",
        "//src/metrics",
        "//src/parse",
        "//src/parse/asp",
        "//src/remote",
        "//src/test",
    ],
)

go_test(
    name = "plz_test",
    srcs = glob(["*_test.go"]),
    deps = [
        ":plz",
        "///third_party/go/github.com_stretchr_testify//assert",
        "///third_party/go/github.com_stretchr_testify//require",
        "//src/cli",
        "//src/core",
    ],
)
