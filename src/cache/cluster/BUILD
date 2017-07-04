go_library(
    name = 'cluster',
    srcs = glob(['*.go'], excludes = ['*_test.go']),
    deps = [
        '//src/cache/proto:rpc_cache',
        '//src/cache/tools',
        '//third_party/go:grpc',
        '//third_party/go:logging',
        '//third_party/go:memberlist',
    ],
    visibility = ['//src/cache/server:all'],
)

go_test(
    name = 'cluster_test',
    srcs = ['cluster_test.go'],
    container = True,
    static = True,
    deps = [
        ':cluster',
        '//src/cache/proto:rpc_cache',
        '//src/cache/tools',
        '//third_party/go:context',
        '//third_party/go:grpc',
        '//third_party/go:testify',
    ],
)
