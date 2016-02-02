# Tests on the proto rules.

proto_library(
    name = 'test_proto',
    srcs = ['test.proto'],
)

python_test(
    name = 'specific_out_test',
    srcs = ['specific_out_test.py'],
    deps = [
        ':test_proto',
    ],
)
