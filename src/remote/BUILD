go_library(
    name = 'remote',
    srcs = [
        'grpc_client.go',
        'grpc_server.go',
        'marshalling.go',
        'resources.go',
    ],
    deps = [
        '//src/core',
        '//src/output',
        '//src/remote/proto:build_event',
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
        ':remote',
        '//src/cli',
        '//src/core',
        '//third_party/go:testify',
    ],
)
