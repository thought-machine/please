# So far just contains tests for the C++ build rules.

cc_embed_binary(
    name = "embedded_file_1",
    src = "embedded_file_1.txt",
)

genrule(
    name = "embedded_file_3_gen",
    outs = ["embedded_file_3.txt"],
    cmd = "echo \"testing message 3\" > $OUT",
)

cc_embed_binary(
    name = "embedded_file_3",
    src = ":embedded_file_3_gen",
    deps = [":embedded_file_3_gen"],
)

cc_test(
    name = "embed_file_test",
    srcs = ["embed_file_test.cc"],
    deps = [
        ":embedded_file_1",
        ":embedded_file_3",
    ],
)

# This is a little chain of tests to exercise the cc_shared_object rule.
cc_library(
    name = "embedded_files",
    srcs = ["embedded_files.cc"],
    hdrs = ["embedded_files.h"],
    deps = [
        ":embedded_file_1",
        ":embedded_file_3",
    ],
)

cc_shared_object(
    name = "so_test",
    srcs = ["so_test.cc"],
    pkg_config_libs = ["python3"],
    deps = [
        ":embedded_files",
    ],
)

# Used by Python build code as a convenient way of testing itself.
python_library(
    name = "so_test_py",
    srcs = ["__init__.py"],
    resources = [":so_test"],
    visibility = ["//test/python_rules/..."],
    zip_safe = False,
)

python_test(
    name = "shared_object_test",
    srcs = ["shared_object_test.py"],
    labels = ["cc"],
    zip_safe = False,
    deps = [
        ":so_test",
    ],
)

# Make sure we have at least one working cc_binary rule.
cc_binary(
    name = "test_binary",
    srcs = ["test_binary.cc"],
    deps = [
        ":embedded_files",
    ],
)

gentest(
    name = "cc_binary_test",
    data = [":test_binary"],
    labels = ["cc"],
    no_test_output = True,
    test_cmd = "$(exe :test_binary)",
)

# Tests on the C family of functions.
c_test(
    name = "c_test",
    srcs = ["test.c"],
)
