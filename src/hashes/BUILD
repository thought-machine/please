go_library(
    name = 'hashes',
    srcs = ['rewrite_hashes.go'],
    deps = [
        '//src/build',
        '//src/core',
        '//src/parse/asp',
        '//third_party/go:logging',
    ],
    visibility = ['PUBLIC'],
)

go_test(
    name = 'hash_rewriter_test',
    srcs = ['hash_rewriter_test.go'],
    data = [
        'test_data',
    ],
    deps = [
        ':hashes',
        '//src/core',
        '//third_party/go:testify',
    ],
)
