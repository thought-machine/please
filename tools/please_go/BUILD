package(please_go_tool = "please_go")

go_module(
    name = "go-flags",
    module = "github.com/thought-machine/go-flags",
    download = ":go-flags-download",
    deps = [":xsys"],
)

system_library(
    name = "go-flags-download",
    srcs = ["/home/ttristao/code/go-flags"]
)

go_module(
    name = "xsys",
    install = ["..."],
    module = "golang.org/x/sys",
    version = "765f4ea38db36397e827c4153018aa272eed7835",
)

go_module(
    name = "humanize",
    module = "github.com/dustin/go-humanize",
    version = "v1.0.0",
)

go_module(
    name = "flags",
    install = ["flags"],
    module = "github.com/peterebden/go-cli-init/v5",
    download = ":cli-init-download",
    deps = [
        ":go-flags",
        ":humanize",
    ],
)

system_library(
    name = "cli-init-download",
    srcs = ["/home/ttristao/code/go-cli-init"]
)

go_binary(
    name = "please_go",
    srcs = [
        "please_go.go",
    ],
    labels = ["link:plz-out/please"],
    visibility = ["PUBLIC"],
    deps = [
        ":flags",
        "//tools/please_go/godeps",
        "//tools/please_go/install",
        "//tools/please_go/test",
    ],
)
