system_library(
    name = 'source_context_proto',
    srcs = ['/usr/include/google/protobuf/source_context.proto'],
)

proto_library(
    name = 'source_context',
    srcs = [':source_context_proto'],
)

python_test(
    name = 'system_dep_test',
    srcs = ['system_dep_test.py'],
    labels = ['proto'],
    deps = [
        ':source_context',
    ],
)

go_test(
    name = 'go_system_dep_test',
    srcs = ['system_dep_test.go'],
    labels = ['proto'],
    deps = [
        ':source_context',
    ],
)
