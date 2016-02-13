filegroup(
    name = 'wrapper_script',
    srcs = ['pleasew.sh'],
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
    deps = [
        ':misc',
        '//src/output',
    ],
    visibility = ['PUBLIC'],
)

go_test(
    name = 'plz_diff_graphs_test',
    srcs = ['plz_diff_graphs_test.go'],
    deps = [
        ':misc',
        '//src/core',
        '//third_party/go:testify',
    ],
    data = glob(['test_data/*.json']),
)
