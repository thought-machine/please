go_binary(
    name = "sandbox",
    srcs = glob(["*.go"]),
    deps = [
        "//third_party/go:xsys",
    ],
)

# go_binary() provides the lib rule. This makes us depend on the actual binary instead of that in go_library()
filegroup(
    name = "sandbox_tool",
    srcs = [":sandbox"],
    visibility = ["//src/sandbox/..."],
)

genrule(
    name = "sandbox_hash",
    srcs = [":sandbox"],
    visibility = ["//src/sandbox/..."],
    cmd = "sha256sum $SRC > $OUT",
    outs = ["sandbox.sha256"],
)