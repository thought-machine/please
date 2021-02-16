go_library(
    name = "embed",
    srcs = ["embed.go"],
    visibility = ["//tools/please_go_embed/..."],
)

go_test(
    name = "embed_test",
    srcs = ["embed_test.go"],
    data = ["test_data"],
    deps = [
        ":embed",
        "//third_party/go:testify",
    ],
)
