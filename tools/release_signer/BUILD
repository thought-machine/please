go_binary(
    name = "release_signer",
    srcs = ["main.go"],
    static = CONFIG.OS == "linux",
    visibility = [
        "//package/...",
        "//tools/javac_worker:all",
    ],
    deps = [
        "//src/cli",
        "//tools/release_signer/signer",
        "///third_party/go/github.com_sigstore_sigstore_pkg_signature_kms_gcp//:gcp",
    ],
)
