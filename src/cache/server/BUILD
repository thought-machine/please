go_library(
    name = 'server',
    srcs = ['http_server.go', 'rpc_server.go', 'cache.go'],
    deps = [
        '//src/cache/proto:rpc_cache',
        '//src/cache/tools',
        '//src/core',
        '//third_party/go:mux',
        '//third_party/go:logging',
        '//third_party/go:humanize',
        '//third_party/go:grpc',
    ],
    # Exposed for a test only.
    visibility = ['//src/cache/...'],
)

go_binary(
    name = 'http_cache_server_bin',
    main = 'http_server_main.go',
    deps = [
        ':server',
        '//src/output',
        '//third_party/go:logging',
    ],
    visibility = ['PUBLIC'],
)

go_binary(
    name = 'rpc_cache_server_bin',
    main = 'rpc_server_main.go',
    deps = [
        ':server',
        '//src/output',
        '//third_party/go:logging',
    ],
    visibility = ['PUBLIC'],
)

go_test(
    name = 'http_server_test',
    srcs = ['http_server_test.go'],
    deps = [
        ':server',
    ],
    data = [':test_data'],
)

go_test(
    name = 'rpc_server_test',
    srcs = ['rpc_server_test.go'],
    deps = [
        ':server',
        '//third_party/go:testify',
    ],
    data = [':test_data', '//src/cache:test_data'],
    container = True,
)

go_test(
    name = 'cache_test',
    srcs = ['cache_test.go'],
    deps = [
        ':server',
        '//third_party/go:testify',
    ],
    data = [':test_data'],
)

filegroup(
    name = 'test_data',
    srcs = glob(['test_data/**']),
    test_only = True,
)
