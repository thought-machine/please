go_library(
    name = "plz",
    srcs = ["plz.go"],
    visibility = ["PUBLIC"],
    deps = [
        "///third_party/go/github.com_peterebden_go-cli-init_v5//flags",
        "//src/build",
        "//src/cli",
        "//src/cli/logging",
        "//src/core",
        "//src/fs",
        "//src/metrics",
        "//src/parse",
        "//src/remote",
        "//src/test",
    ],
)

go_test(
    name = "plz_test",
    srcs = ["plz_test.go"],
    deps = [
        ":plz",
        "///third_party/go/github.com_stretchr_testify//assert",
        "//src/cli",
        "//src/core",
    ],
)
