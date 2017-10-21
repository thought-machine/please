go_library(
    name = 'diff',
    srcs = ['plz_diff_graphs.go'],
    deps = [
        '//src/core',
        '//src/query',
        '//third_party/go:logging',
    ],
    visibility = ['//tools/please_diff_graphs'],
)

go_test(
    name = 'plz_diff_graphs_test',
    srcs = ['plz_diff_graphs_test.go'],
    data = glob(['test_data/*.json']),
    deps = [
        ':diff',
        '//src/core',
        '//third_party/go:testify',
    ],
)
