go_binary(
    name = "please_shim",
    srcs = ["main.go"],
    definitions = {
        # Enforce this pseudo-version instead of "0.0.0" to not conflict with
        # an empty please version config.
        "github.com/thought-machine/please/src/core.PleaseVersion": "0.0.1",
    },
    visibility = [
        "//package:all",
        "//test/please_shim/...",
    ],
    deps = [
        "//src/cli",
        "//src/core",
        "//src/fs",
        "//src/update",
        "//third_party/go:go-flags",
        "//third_party/go:logging",
    ],
)
