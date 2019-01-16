go_library(
    name = "tags",
    srcs = ["tags1.go", "tags2.go"],
    tags = ["test"],
)

go_test(
    name = "tags_test",
    srcs = ["tags_test.go"],
    tags = ["test"],
    deps = [
        ":tags",
        "//third_party/go:testify",
    ],
)
