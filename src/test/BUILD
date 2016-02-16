go_library(
    name = 'test',
    srcs = glob(['*.go'], excludes=['*_test.go']),
    deps = [
        '//src/core',
	'//src/build',
	'//third_party/go:cover',
	'//third_party/go:logging',
    ],
    visibility = ['PUBLIC'],
)

go_test(
    name = 'results_test',
    srcs = ['results_test.go'],
    deps = [
        ':test',
    ],
    data = glob(['test_data/*'])
)

go_test(
    name = 'test_step_test',
    srcs = ['test_step_test.go'],
    deps = [
        ':test',
        '//third_party/go:testify',
    ],
)

go_test(
    name = 'container_test',
    srcs = ['container_test.go'],
    data = ['test_data/container_data.txt'],
    container = True
)

go_test(
    name = 'container_args_test',
    srcs = ['container_args_test.go'],
    container = {
        'docker_run_args': '-e TEST_VALUE=WIBBLE',
    }
)

go_test(
    name = 'coverage_test',
    srcs = ['coverage_test.go'],
    data = [
        'test_data/python-coverage.xml',
        'test_data/go_coverage.txt',
        'test_data/go_coverage_2.txt',
        'test_data/go_coverage_3.txt',
    ],
    deps = [
        ':test',
    ],
)

# This rule tests the no_test_output flag. If that isn't honoured
# plz would report a test failure because results were missing.
gentest(
    name = 'no_test_output_test',
    test_cmd = 'echo SUCCESS',
    no_test_output = True
)

# This tests that data files exist in the correct location, and
# indirectly performs a basic test of sh_test which we don't use elsewhere.
sh_test(
    name = 'data_files_test',
    src = 'data_files_test.sh',
    data = ['test_data/container_data.txt'],
)

# This test is here as a convenience to test the flakiness functionality.
# It's just using random internally so won't pass consistently.
python_test(
    name = 'flakiness_test',
    srcs = ['flakiness_test.py'],
    flaky = 5,
    labels = ['manual'],
)
