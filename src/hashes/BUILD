go_library(
    name = "hashes",
    srcs = ["rewrite_hashes.go"],
    visibility = ["PUBLIC"],
    deps = [
        "//src/cli/logging",
        "//src/core",
        "//src/parse/asp",
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
        "///third_party/go/github.com_stretchr_testify//assert",
        "//src/core",
        "//src/fs",
    ],
)
