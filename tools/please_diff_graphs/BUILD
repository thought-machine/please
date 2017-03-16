go_library(
    name = 'diff_graphs',
    srcs = ['plz_diff_graphs.go'],
    out = 'please_diff_graphs.a',
    deps = [
        '//src/core',
        '//src/query',
        '//third_party/go:logging',
    ],
)

go_binary(
    name = 'please_diff_graphs',
    srcs = ['plz_diff_graphs_main.go'],
    deps = [
        ':diff_graphs',
        '//src/cli',
    ],
    visibility = ['PUBLIC'],
)

go_test(
    name = 'plz_diff_graphs_test',
    srcs = ['plz_diff_graphs_test.go'],
    data = glob(['test_data/*.json']),
    deps = [
        ':diff_graphs',
        '//src/core',
        '//third_party/go:testify',
    ],
)
