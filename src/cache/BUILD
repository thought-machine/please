# Currently gRPC is not supported on FreeBSD, so build the stub version of the cache
# without it implemented.
# TODO(pebers): Remove once upstream resolves https://github.com/grpc/grpc/issues/1035.
if CONFIG.OS == 'freebsd':
    go_library(
        name = 'cache',
        srcs = glob(['*.go'], excludes=['*_test.go', 'rpc_cache.go']),
        deps = [
            '//src/core',
            '//third_party/go:logging',
        ],
        visibility = ['PUBLIC'],
    )

else:
    go_library(
        name = 'cache',
        srcs = glob(['*.go'], excludes=['*_test.go', 'rpc_cache_stub.go']),
        deps = [
            '//src/core',
            '//src/cache/proto:rpc_cache',
            '//src/cache/tools',
            '//third_party/go:logging',
            '//third_party/go:grpc',
        ],
        visibility = ['PUBLIC'],
    )

    go_test(
        name = 'rpc_cache_test',
        srcs = ['rpc_cache_test.go'],
        deps = [
            ':cache',
            '//src/cache/server',
            '//third_party/go:grpc',
            '//third_party/go:logging',
            '//third_party/go:testify',
        ],
        data = [':test_data'],
        container = True,  # Brings up an internal server
    )

filegroup(
    name = 'test_data',
    srcs = ['test_data'],
    test_only = True,
    visibility = ['//src/cache/...'],
)

go_test(
    name = 'http_cache_test',
    srcs = ['http_cache_test.go'],
    container = True,  # Brings up an internal server
    data = [':test_data'],
    static = True,
    deps = [
        ':cache',
        '//src/cache/server',
        '//third_party/go:logging',
    ],
)

go_test(
    name = 'async_cache_test',
    srcs = ['async_cache_test.go'],
    deps = [
        ':cache',
        '//third_party/go:testify',
    ],
)
