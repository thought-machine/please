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
        "///third_party/go/github.com_thought-machine_go-flags//:go-flags",
        "///third_party/go/gopkg.in_op_go-logging.v1//:go-logging.v1",
        "//src/cli",
        "//src/core",
        "//src/fs",
        "//src/update",
    ],
)
