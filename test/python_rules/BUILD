# Test for a panic provoked by a python_library using a proto_library as a resource.
proto_library(
    name = 'test_proto',
    srcs = ['test.proto'],
)

python_library(
    name = 'proto_lib',
    srcs = ['proto_lib.py'],
    resources = [':test_proto'],
)

python_test(
    name = 'proto_test',
    srcs = ['proto_test.py'],
    deps = [':proto_lib'],
)

python_test(
    name = 'strip_source_test',
    srcs = ['strip_source_test.py'],
    labels = ['manual'],  # only works in stripped configuration
)
