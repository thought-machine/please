go_library(
    name = "toolchain",
    srcs = ["toolchain.go"],
    visibility = ["//tools/please_go/..."],
    deps = ["//tools/please_go/install/exec"],
)

go_test(
    name = "toolchain_test",
    srcs = ["toolchain_test.go"],
    data = ["//third_party/go:toolchain|go"],
    labels = ["no-musl"],
    deps = [
        ":toolchain",
        "///third_party/go/github.com_stretchr_testify//require",
    ],
)
