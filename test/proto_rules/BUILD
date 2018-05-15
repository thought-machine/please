# Tests on the proto rules.

# Test that we can use a 'generated' source as well as a real one.
filegroup(
    name = "gen_proto",
    srcs = ["test.proto"],
)

proto_library(
    name = "test_proto",
    srcs = [":gen_proto"],
    visibility = ["//test/misc_rules:all"],  # Used by another test over here too.
)

python_test(
    name = "specific_out_test",
    srcs = ["specific_out_test.py"],
    deps = [
        ":test_proto",
    ],
)

proto_library(
    name = "js_test_proto",
    srcs = ["js_test.proto"],
    languages = [
        "js",
    ],
)

filegroup(
    name = "js_test_proto_filegroup",
    srcs = [":js_test_proto"],
    requires = ["js"],
)

python_test(
    name = "js_out_test",
    srcs = ["js_out_test.py"],
    data = [
        ":js_test_proto_filegroup",
    ],
)
