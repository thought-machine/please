go_library(
    name = 'server',
    srcs = [
        'cache.go',
        'http_server.go',
        'rpc_server.go',
    ],
    deps = [
        '//src/cache/cluster',
        '//src/cache/proto:rpc_cache',
        '//src/core',
        '//third_party/go:atime',
        '//third_party/go:concurrent-map',
        '//third_party/go:grpc',
        '//third_party/go:humanize',
        '//third_party/go:logging',
        '//third_party/go:mux',
    ],
    # Exposed for a test only.
    visibility = ['//src/cache/...'],
)

go_binary(
    name = 'http_cache_server_bin',
    main = 'http_server_main.go',
    deps = [
        ':server',
        '//src/cli',
        '//third_party/go:logging',
    ],
    visibility = ['PUBLIC'],
)

go_binary(
    name = 'rpc_cache_server_bin',
    main = 'rpc_server_main.go',
    deps = [
        ':server',
        '//src/cache/cluster',
        '//src/cli',
        '//third_party/go:logging',
    ],
    visibility = ['PUBLIC'],
)

go_test(
    name = 'http_server_test',
    srcs = ['http_server_test.go'],
    data = [':test_data'],
    deps = [
        ':server',
    ],
)

go_test(
    name = 'rpc_server_test',
    srcs = ['rpc_server_test.go'],
    container = True,
    data = [
        ':test_data',
        '//src/cache:test_data',
    ],
    deps = [
        ':server',
        '//third_party/go:testify',
    ],
)

go_test(
    name = 'cache_test',
    srcs = ['cache_test.go'],
    data = [':test_data'],
    deps = [
        ':server',
        '//src/core',
        '//third_party/go:testify',
    ],
)

go_test(
    name = 'cache_stress_test',
    srcs = ['cache_stress_test.go'],
    deps = [
        ':server',
        '//third_party/go:testify',
    ],
)

filegroup(
    name = 'test_data',
    srcs = glob(['test_data/**']),
    test_only = True,
)
