go_binary(
    name = 'http_cache_server',
    srcs = ['http_server_main.go'],
    deps = [
        '//src/cli',
        '//third_party/go:logging',
        '//tools/cache/server',
    ],
    visibility = ['PUBLIC'],
)

go_binary(
    name = 'rpc_cache_server',
    srcs = ['rpc_server_main.go'],
    deps = [
        '//src/cli',
        '//third_party/go:grpc-prometheus',
        '//third_party/go:logging',
        '//third_party/go:prometheus',
        '//tools/cache/cluster',
        '//tools/cache/server',
    ],
    visibility = ['PUBLIC'],
)

go_binary(
    name = 'cleaner',
    srcs = ['cache_cleaner.go'],
    deps = [
        '//src/cli',
        '//third_party/go:atime',
        '//third_party/go:humanize',
        '//third_party/go:logging',
    ],
    visibility = ['PUBLIC'],
)

go_test(
    name = 'cache_cleaner_test',
    srcs = [
        'cache_cleaner.go',
        'cache_cleaner_test.go',
    ],
    deps = [
        '//src/cli',
        '//third_party/go:atime',
        '//third_party/go:humanize',
        '//third_party/go:logging',
        '//third_party/go:testify',
    ],
)
