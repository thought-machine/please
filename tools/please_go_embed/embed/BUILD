go_library(
    name = "embed",
    srcs = ["embed.go"],
    visibility = ["//tools/..."],
)

go_test(
    name = "embed_test",
    srcs = ["embed_test.go"],
    data = ["test_data"],
    deps = [
        ":embed",
        "///third_party/go/github.com_stretchr_testify//assert",
    ],
)
