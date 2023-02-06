subinclude("//test/build_defs")

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
    name = "namespaced_packages_test",
    srcs = ["namespaced_packages_test.py"],
    labels = [
        "py3",
        "pip",
    ],
    deps = [
        "//third_party/python:googleapis_common_protos",
        "//third_party/python:protobuf_pip",
    ],
)

python_test(
    name = "worker_test",
    srcs = ["worker_test.py"],
    worker = "//test/workers:worker",
)

python_test(
    name = "importlib_metadata_test",
    srcs = ["importlib_metadata_test.py"],
    labels = [
        "py3",
        "pip",
    ],
    deps = ["//third_party/python:importlib_metadata"],
)

python_test(
    name = "data_dict_test",
    srcs = ["data_dict_test.py"],
    data = {"txt": ["data.txt"]},
    labels = ["py3"],
)

python_test(
    name = "cx_oracle_darwin_build_test",
    srcs = ["cx_oracle_darwin_build_test.py"],
    labels = [
        "py3",
        "pip",
    ],
    deps = ["//third_party/python:cx_oracle"],
)
