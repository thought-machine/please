go_binary(
    name = "release_signer",
    srcs = ["main.go"],
    visibility = [
        "//package/...",
        "//tools/javac_worker:all",
    ],
    deps = [
        "///third_party/go/github.com_sigstore_sigstore_pkg_signature_kms_gcp//:gcp",
        "//src/cli",
        "//tools/release_signer/signer",
    ],
)
