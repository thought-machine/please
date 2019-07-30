go_library(
    name = "tools",
    srcs = ["hash.go"],
    visibility = [
        "//src/cache/...",
    ],
)

go_test(
    name = "hash_test",
    srcs = ["hash_test.go"],
    deps = [
        ":tools",
        "//third_party/go:testify",
    ],
)
