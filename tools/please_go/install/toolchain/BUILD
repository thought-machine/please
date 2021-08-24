go_library(
    name = "toolchain",
    srcs = [
        "toolchain.go",
    ],
    visibility = ["//tools/please_go/install/..."],
    deps = [
        "//tools/please_go/install/exec",
    ],
)

go_test(
    name = "toolchain_test",
    srcs = ["toolchain_test.go"],
    deps = [
        "//third_party/go:testify",
        ":toolchain",
    ],
    data = ["//third_party/go:toolchain|go"],
)