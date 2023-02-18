go_library(
    name = "embed",
    srcs = ["embed.go"],
    resources = [
        "hello.txt",
        "test_data",
    ],
)

go_test(
    name = "embed_test",
    srcs = ["embed_test.go"],
    resources = [
        "hello.txt",
        "test_data",
    ],
    deps = [
        ":embed",
        "///third_party/go/github.com_stretchr_testify//assert",
    ],
)
