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
    visibility = ['//src/cache/...'],
    test_only = True,
)

go_test(
    name = 'http_cache_test',
    srcs = ['http_cache_test.go'],
    deps = [
        ':cache',
        '//src/cache/server',
        '//third_party/go:logging',
    ],
    data = ['test_data/linux_amd64', 'test_data/darwin_amd64'],
    container = True,  # Brings up an internal server
)
