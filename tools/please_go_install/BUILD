go_mod_download(
    name = "go_flags_srcs",
    module = "github.com/thought-machine/go-flags",
    version = "v1.5.0",
    please_go_install = False,
)

go_mod_download(
    name = "xsys_srcs",
    module = "golang.org/x/sys",
    version = "765f4ea38db36397e827c4153018aa272eed7835",
    please_go_install = False,
)

go_get(
    name = "go-flags",
    get = [],
    install = [
        "github.com/thought-machine/go-flags",
        "golang.org/x/sys/...",
    ],
    deps = [
        ":go_flags_srcs",
        ":xsys_srcs",
    ],
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
