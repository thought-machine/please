system_library(
    name = 'source_context_proto',
    srcs = ['/usr/include/google/protobuf/source_context.proto'],
)

proto_library(
    name = 'source_context',
    srcs = [':source_context_proto'],
    visibility = ['//test/system_deps/...'],
)

python_test(
    name = 'system_dep_test',
    srcs = ['system_dep_test.py'],
    labels = ['proto'],
    deps = [
        ':source_context',
    ],
)
