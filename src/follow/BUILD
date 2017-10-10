go_library(
    name = 'follow',
    srcs = [
        'grpc_client.go',
        'grpc_server.go',
        'marshalling.go',
        'resources.go',
    ],
    deps = [
        '//src/core',
        '//src/follow/proto:build_event',
        '//src/output',
        '//third_party/go:context',
        '//third_party/go:grpc',
        '//third_party/go:logging',
        '//third_party/go:psutil',
    ],
    visibility = ['PUBLIC'],
)

go_test(
    name = 'grpc_test',
    srcs = ['grpc_test.go'],
    deps = [
        ':follow',
        '//src/cli',
        '//src/core',
        '//third_party/go:testify',
    ],
)
