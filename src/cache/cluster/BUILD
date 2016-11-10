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
