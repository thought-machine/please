system_library(
    name = 'timestamp_proto',
    srcs = ['/usr/include/google/protobuf/timestamp.proto'],
)

proto_library(
    name = 'timestamp',
    srcs = [':timestamp_proto'],
)

python_test(
    name = 'system_dep_test',
    srcs = ['system_dep_test.py'],
    deps = [
        ':timestamp',
    ],
)

go_test(
    name = 'go_system_dep_test',
    srcs = ['system_dep_test.go'],
    deps = [
        ':timestamp',
    ],
)
