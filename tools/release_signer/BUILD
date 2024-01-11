go_binary(
    name = "release_signer",
    srcs = ["main.go"],
    static = CONFIG.OS == "linux",
    visibility = [
        "//package/...",
        "//tools/javac_worker:all",
    ],
    deps = [
        "///third_party/go/github.com_sigstore_sigstore//pkg/signature/kms/gcp",
        "//src/cli",
        "//tools/release_signer/signer",
    ],
)
