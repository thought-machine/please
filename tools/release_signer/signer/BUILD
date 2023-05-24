go_library(
    name = "signer",
    srcs = ["signer.go"],
    visibility = ["//tools/release_signer"],
    deps = [
        "///third_party/go/github.com_ProtonMail_go-crypto//openpgp",
        "///third_party/go/github.com_sigstore_sigstore//pkg/signature",
    ],
)

go_test(
    name = "signer_test",
    srcs = ["signer_test.go"],
    data = ["test_data"],
    deps = [
        ":signer",
        "///third_party/go/github.com_ProtonMail_go-crypto//openpgp",
        "///third_party/go/github.com_sigstore_sigstore//pkg/cryptoutils",
        "///third_party/go/github.com_sigstore_sigstore//pkg/signature",
        "///third_party/go/github.com_stretchr_testify//assert",
    ],
)
