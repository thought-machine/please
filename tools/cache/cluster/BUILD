go_library(
    name = 'cluster',
    srcs = ['cluster.go'],
    deps = [
        '//src/cache/proto:rpc_cache',
        '//src/cache/tools',
        '//third_party/go:grpc',
        '//third_party/go:logging',
        '//third_party/go:memberlist',
    ],
    visibility = ['//tools/cache/...'],
)

go_test(
    name = 'cluster_test',
    srcs = ['cluster_test.go'],
    flaky = True,  # Something looks not 100% synced properly here. TODO(peterebden): fix.
    deps = [
        ':cluster',
        '//src/cache/proto:rpc_cache',
        '//src/cache/tools',
        '//third_party/go:context',
        '//third_party/go:grpc',
        '//third_party/go:testify',
    ],
)
