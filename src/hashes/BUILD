go_library(
    name = "hashes",
    srcs = ["rewrite_hashes.go"],
    pgo_file = "//:pgo",
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
    data = [":hash_rewriter_test_data"],
    # So //test/windows can run this same binary, cross-built, under Wine.
    visibility = ["//test/windows:all"],
    deps = [
        ":hashes",
        "///third_party/go/github.com_stretchr_testify//assert",
        "//src/core",
        "//src/fs",
    ],
)

# Exposed so //test/windows can put it beside the test binary when running it under Wine.
filegroup(
    name = "hash_rewriter_test_data",
    srcs = ["test_data"],
    test_only = True,
    visibility = ["//test/windows:all"],
)
