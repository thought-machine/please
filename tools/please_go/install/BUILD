# We want to work out the of the test_data dir and this makes it slightly easier
package(go_test_root_compat = True)

go_library(
    name = "install",
    srcs = [
        "install.go",
    ],
    visibility = ["PUBLIC"],
    deps = [
        "//tools/please_go/install/exec",
        "//tools/please_go/install/toolchain",
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
        "//src/fs",
        "//third_party/go:testify",
    ],
)
