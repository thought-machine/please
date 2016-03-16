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

go_test(
    name = 'rpc_cache_test',
    srcs = ['rpc_cache_test.go'],
    deps = [
        ':cache',
        '//src/cache/server',
        '//third_party/go:logging',
    ],
    data = ['test_data/linux_amd64', 'test_data/darwin_amd64', 'test_data/plz-out'],
    container = True,  # Brings up an internal server
)
