go_module(
    name = "go-flags",
    module = "github.com/thought-machine/go-flags",
    version = "v1.6.0",
    deps = [":xsys"],
)

go_module(
    name = "xsys",
    install = ["..."],
    module = "golang.org/x/sys",
    version = "039c03cc5b867cd7b06a19ff375be5c945c80b10",
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
    version = "v5.1.0",
    deps = [
        ":go-flags",
        ":humanize",
    ],
)

go_binary(
    name = "please_go",
    srcs = ["please_go.go"],
    labels = ["link:plz-out/please"],
    visibility = ["PUBLIC"],
    deps = [
        "///third_party/go/github.com_peterebden_go-cli-init_v5//flags",
        "//tools/please_go/godeps",
        "//tools/please_go/install",
        "//tools/please_go/test",
    ],
)
