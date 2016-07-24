filegroup(
    name = 'wrapper_script',
    srcs = ['pleasew'],
    visibility = ['PUBLIC'],
)

go_library(
    name = 'misc',
    srcs = ['plz_diff_graphs.go'],
    deps = [
        '//src/core',
        '//src/query',
        '//third_party/go:logging',
    ],
)

go_binary(
    name = 'plz_diff_graphs',
    main = 'plz_diff_graphs_main.go',
    visibility = ['PUBLIC'],
    deps = [
        ':misc',
        '//src/output',
    ],
)

go_test(
    name = 'plz_diff_graphs_test',
    srcs = ['plz_diff_graphs_test.go'],
    data = glob(['test_data/*.json']),
    deps = [
        ':misc',
        '//src/core',
        '//third_party/go:testify',
    ],
)
