go_binary(
    name = "release_signer",
    srcs = ["main.go"],
    visibility = [
        "//package/...",
        "//tools/javac_worker:all",
    ],
    deps = [
        "//src/cli",
        "//third_party/go:sigstore",
        "//tools/release_signer/signer",
    ],
)
