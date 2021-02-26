package(go_install_tool = "please_go_install")

go_module(
    name = "go-flags",
    module = "github.com/thought-machine/go-flags",
    version = "v1.5.0",
    deps = [":xsys"],
)

go_module(
    name = "xsys",
    install = ["..."],
    module = "golang.org/x/sys",
    version = "765f4ea38db36397e827c4153018aa272eed7835",
)


go_binary(
    name = "please_go_install",
    srcs = [
        "please_go_install.go",
    ],
    labels = ["link:plz-out/please"],
    visibility = ["PUBLIC"],
    deps = [
        ":go-flags",
        "//tools/please_go_install/exec",
        "//tools/please_go_install/toolchain",
    ],
)

go_test(
    name = "please_go_install_test",
    srcs = ["please_go_install_test.go"],
    data = {
        "pleaes_go_install": [":please_go_install"],
        "test_data": ["test_data"],
    },
    deps = [
        "//src/fs",
        "//third_party/go:testify",
    ],
)
