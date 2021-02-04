#TODO(jpoole): once bugs are fixed, make this use the please_go_install in ~/.please
package(FF_PLEASE_GO_INSTALL = False)

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
        "main.go",
    ],
    visibility = ["PUBLIC"],
    deps = [
        ":go-flags",
        "//tools/please_go_install/exec",
        "//tools/please_go_install/toolchain",
    ],
)
