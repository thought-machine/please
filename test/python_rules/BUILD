# Test for a panic provoked by a python_library using a proto_library as a resource.
proto_library(
    name = "test_proto",
    srcs = ["test.proto"],
    languages = ["py"],
)

python_library(
    name = "proto_lib",
    srcs = ["proto_lib.py"],
    resources = [":test_proto"],
)

python_test(
    name = "proto_test",
    srcs = ["proto_test.py"],
    deps = [":proto_lib"],
)

python_binary(
    name = "strip_source",
    main = "strip_source.py",
    strip = True,
)

python_test(
    name = "strip_source_test",
    srcs = ["strip_source_test.py"],
    data = [":strip_source"],
    labels = ["py3"],
)

# Test that has no actual tests in it. Should still count as a pass.
python_test(
    name = "empty_test",
    srcs = ["empty_test.py"],
)

# Test that the build rules accept a python_binary that has a rule as the main.
filegroup(
    name = "rule_main",
    srcs = ["rule_main.py"],
)

python_binary(
    name = "rule_main_pex",
    main = ":rule_main",
)

sh_test(
    name = "rule_main_test",
    src = "rule_main_test.sh",
    data = [":rule_main_pex"],
    labels = ["python3"],
)

# Used to test zip-safety flags.
python_test(
    name = "zip_unsafe_test",
    srcs = ["zip_unsafe_test.py"],
    labels = [
        "cc",  # Depends on cc rules to build
        "py3_pkg_config",  # and pkg-config python3
    ],
    deps = [
        "//test/cc_rules/gcc:so_test_py",
    ],
)

# Test flags on python_test
python_test(
    name = "flags_test",
    srcs = ["flags_test.py"],
    flags = "--test_flag",
)

# Tests for a subtle case where python_test rules should get a pex when
# specified as data, but not when depending directly.
python_binary(
    name = "data_dep",
    main = "data_dep.py",
)

python_test(
    name = "data_dep_test",
    srcs = ["data_dep_test.py"],
    data = [":data_dep"],
    deps = [":data_dep"],
)

python_test(
    name = "data_dep_only_test",
    srcs = ["data_dep_only_test.py"],
    data = [":data_dep"],
)

python_library(
    name = "python_coverage",
    srcs = ["python_coverage.py"],
    labels = [
        "py2",
        "py3",
    ],
)

python_test(
    name = "python2_coverage_test",
    srcs = ["python_coverage_test.py"],
    interpreter = "python2",
    labels = [
        "py2",
        "py3",
    ],
    deps = [":python_coverage"],
)

python_test(
    name = "python3_coverage_test",
    srcs = ["python_coverage_test.py"],
    labels = ["py3"],
    deps = [":python_coverage"],
)

python_test(
    name = "unittest_unicode_test",
    srcs = ["unittest_unicode_test.py"],
)

python_test(
    name = "numpy_test",
    srcs = ["numpy_test.py"],
    labels = [
        "py3",
        "pip",
    ],
    deps = ["//third_party/python:numpy"],
)

python_test(
    name = "tensorflow_test",
    srcs = ["tensorflow_test.py"],
    labels = [
        "py3",
        "pip",
    ],
    deps = ["//third_party/python:tensorflow"],
)

python_test(
    name = "pandas_test",
    srcs = ["pandas_test.py"],
    labels = [
        "py3",
        "pip",
    ],
    deps = ["//third_party/python:pandas"],
)

python_test(
    name = "scipy_test",
    srcs = ["scipy_test.py"],
    labels = [
        "py3",
        "pip",
    ],
    deps = ["//third_party/python:scipy"],
)

python_test(
    name = "worker_test",
    srcs = ["worker_test.py"],
    worker = "//test/workers:worker",
)
