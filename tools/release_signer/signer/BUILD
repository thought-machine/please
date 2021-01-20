go_library(
    name = "signer",
    srcs = ["signer.go"],
    visibility = ["//tools/release_signer"],
    deps = [
        "//third_party/go:xcrypto",
    ],
)

go_test(
    name = "signer_test",
    srcs = ["signer_test.go"],
    data = ["test_data"],
    deps = [
        ":signer",
        "//third_party/go:testify",
        "//third_party/go:xcrypto",
    ],
)
