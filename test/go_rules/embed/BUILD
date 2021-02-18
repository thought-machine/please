go_library(
    name = "embed",
    srcs = ["embed.go"],
    resources = ["hello.txt"],
)

go_test(
    name = "embed_test",
    srcs = ["embed_test.go"],
    resources = ["hello.txt"],
    deps = [
        ":embed",
        "//third_party/go:testify",
    ],
)
