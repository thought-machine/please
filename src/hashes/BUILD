go_library(
    name = "hashes",
    srcs = ["rewrite_hashes.go"],
    visibility = ["PUBLIC"],
    deps = [
        "//src/core",
        "//src/parse/asp",
        "//third_party/go:logging",
    ],
)

go_test(
    name = "hash_rewriter_test",
    srcs = ["hash_rewriter_test.go"],
    data = [
        "test_data",
    ],
    deps = [
        ":hashes",
        "//src/core",
        "//src/fs",
        "//third_party/go:testify",
    ],
)
