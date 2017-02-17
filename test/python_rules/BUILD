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

# Test that has no actual tests in it. Should still count as a pass.
python_test(
    name = 'empty_test',
    srcs = ['empty_test.py'],
)

# Test that the build rules accept a python_binary that has a rule as the main.
filegroup(
    name = 'rule_main',
    srcs = ['rule_main.py'],
)

python_binary(
    name = 'rule_main_pex',
    main = ':rule_main',
)

sh_test(
    name = 'rule_main_test',
    src = 'rule_main_test.sh',
    data = [':rule_main_pex'],
)

# Used to test zip-safety flags.
python_test(
    name = 'zip_unsafe_test',
    srcs = ['zip_unsafe_test.py'],
    labels = ['cc'],  # Depends on cc rules to build
    deps = [
        '//test/cc_rules/gcc:so_test_py',
    ],
)

# Tests for a subtle case where python_test rules should get a pex when
# specified as data, but not when depending directly.
python_binary(
    name = 'data_dep',
    main = 'data_dep.py',
)

python_test(
    name = 'data_dep_test',
    srcs = ['data_dep_test.py'],
    data = [':data_dep'],
    deps = [':data_dep'],
)

python_test(
    name = 'python2_coverage_test',
    srcs = ['python2_coverage_test.py'],
    interpreter = 'python2',
    labels = ['py2'],
)

python_test(
    name = 'python3_coverage_test',
    srcs = ['python3_coverage_test.py'],
    interpreter = 'python3.5',
    labels = ['py3'],
)
