go_library(
    name = "install",
    srcs = ["install.go"],
    visibility = ["PUBLIC"],
    deps = [
        "//tools/please_go/install/exec",
        "//tools/please_go/install/toolchain",
        "//tools/please_go_embed/embed",
    ],
)

go_test(
    name = "install_test",
    srcs = ["install_test.go"],
    data = {
        "test_data": ["test_data"],
    },
    deps = [
        ":install",
        "///third_party/go/github.com_stretchr_testify//assert",
        "///third_party/go/github.com_stretchr_testify//require",
        "//src/fs",
        "//tools/please_go/install/exec",
    ],
)
