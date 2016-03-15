# A series of end-to-end tests on the Please binary.
#
# These are a little fragile since they assume things about specific output messages, which
# of course we might rather not. However it's something of a pain to get good test coverage
# since by its nature the tool has heaps of side effects, so this is at least one way of
# reassuring ourselves that it does behave as expected.
#
# Note that we have to be kinda careful with this; since it invokes plz to run tests while
# an instance of it is already going, there are potential concurrency issues. These are
# mitigated by having these tests only run tests in this package which are tagged as manual
# so the bootstrap script won't try to run them twice simultaneously.

subinclude('//build_defs:plz_e2e_test')

# Tests the expected output of 'query somepath'.
# Note that you have to be careful with the choice of targets, since the path
# found is not necessarily unique or stable.
plz_e2e_test(
    name = 'query_somepath_test',
    cmd = 'plz query somepath //src/build/python:bootstrap_pexer //third_party/python:coverage',
    expected_output = 'query_somepath_test.txt',
)
plz_e2e_test(
    name = 'query_somepath_reverse_test',
    cmd = 'plz query somepath //third_party/python:coverage //src/build/python:bootstrap_pexer',
    expected_output = 'query_somepath_test.txt',  # Output should be the same as above
)
plz_e2e_test(
    name = 'query_somepath_nopath_test',
    cmd = 'plz query somepath //src:please //third_party/java:guava',
    expected_output = 'query_somepath_nopath_test.txt',
)

# Tests that targets can only use other targets that they depend on.
plz_e2e_test(
    name = 'dep_required_test',
    cmd = 'plz build //test:failed_dep',
    expect_output_contains = "//test:failed_dep can't use //src/core; doesn't depend on target //src/core",
    expected_failure = True,
)
build_rule(
    name = 'failed_dep',
    cmd = 'echo $(location //src/core)',
    labels = ['manual'],
)

# Test that we count test output correctly. Also indirectly tests access to test data files.
# Note that we're stripping coloured output for now. Later we should probably have a flag or something.
plz_e2e_test(
    name = 'test_output_test',
    cmd = 'plz test //test:test_output_test_1 //test:test_output_test_2 | sed "s/\x1B\[[0-9;]*[a-zA-Z]//g"',
    expect_output_contains = '6 tests run; 6 passed.',
    # Invokes a containerised test although it's not itself.
    labels = ['container'],
)
gentest(
    name = 'test_output_test_1',
    data = ['test_output_test_1.txt'],
    test_cmd = 'cp $(location test_output_test_1.txt) test.results',
    labels = ['manual'],
)
gentest(
    name = 'test_output_test_2',
    data = ['test_output_test_2.xml'],
    test_cmd = 'cp $(location test_output_test_2.xml) test.results',
    container = True,
    labels = ['manual'],
)

# Test that on re-running a test it is cached.
plz_e2e_test(
    name = 'test_caching_test',
    cmd = 'plz test //test:caching_test && plz test -v 4 //test:caching_test',
    expect_output_contains = 'Not re-running test //test:caching_test',
)
gentest(
    name = 'caching_test',
    test_cmd = 'true',
    deps = ['//src:please'],
    no_test_output = True,
    labels = ['manual'],
)

# Test that we don't generate coverage on running a test normally (because it's slower).
python_test(
    name = '_no_coverage_output_test',
    srcs = ['coverage_output_test.py'],
    labels = ['manual'],
)
plz_e2e_test(
    name = 'no_coverage_output_test',
    cmd = 'plz test //test:_no_coverage_output_test',
    expect_file_doesnt_exist = '../../../bin/test/.test_coverage__no_coverage_output_test*',
)
# Test that we do generate it when using plz cover.
python_test(
    name = '_coverage_output_test',
    srcs = ['coverage_output_test.py'],
    labels = ['manual'],
)
plz_e2e_test(
    name = 'coverage_output_test',
    cmd = 'plz cover //test:_coverage_output_test',
    expect_file_exists = '../../../bin/test/.test_coverage__coverage_output_test*',
    # Temporarily disabled until #25 is resolved. Until then it recompiles various go_library
    # rules which makes it (and possibly others) flaky.
    labels = ['manual'],
)

# Quick test for plz run
plz_e2e_test(
    name = 'plz_run_test',
    cmd = 'plz run //src:please -- --version',
    expect_output_contains = 'Please version',
)

# Test for query alltargets
plz_e2e_test(
    name = 'query_alltargets_test',
    cmd = 'plz query alltargets',
    expect_output_contains = '//src:please',
)

# Test for query output
plz_e2e_test(
    name = 'query_output_test',
    cmd = 'plz query output //test:query_output_filegroup',
    expected_output = 'query_output_test.txt',
)
filegroup(
    name = 'query_output_filegroup',
    srcs = ['//src:please'],
)

# Test running a test with no-cache
plz_e2e_test(
    name = 'test_nocache_test',
    cmd = 'plz test --nocache //test:nocache_test',
)
gentest(
    name = 'nocache_test',
    test_cmd = 'true',
    deps = ['//src:please'],
    no_test_output = True,
    labels = ['manual'],
)

# Simulates a code generating rule to test the require / provide mechanism.
plz_e2e_test(
    name = 'require_provide_test',
    cmd = 'plz build //test/moar:require_provide_check -v 2 -p',
    expect_output_doesnt_contain = '//test/moar:test_require',
)

# Test for running individual tests
python_test(
    name = 'individual_test_run_py',
    srcs = ['individual_test_run.py'],
    labels = ['manual'],
)
plz_e2e_test(
    name = 'individual_python_test',
    cmd = 'plz test //test:individual_test_run_py TestRunningIndividualTests.test_first_thing',
    expect_output_contains = '1 test target and 1 test run',
)
java_test(
    name = 'individual_test_run_java',
    srcs = ['IndividualTestRun.java'],
    labels = ['manual'],
    deps = [
        '//third_party/java:junit',
    ],
)
plz_e2e_test(
    name = 'individual_java_test',
    cmd = 'plz test //test:individual_test_run_java testFirstThing',
    expect_output_contains = '1 test target and 1 test run',
)
java_test(
    name = 'no_test_run_java',
    srcs = ['NoTestRun.java'],
    labels = ['manual'],
    deps = [
        '//third_party/java:junit',
    ],
)
plz_e2e_test(
    name = 'no_java_test',
    cmd = 'plz test -p //test:no_test_run_java wibblewobble',
    expect_output_contains = '1 failed',
    expected_failure = True,
)

# Test re-runs.
go_test(
    name = '_num_runs_test',
    srcs = ['num_runs_test.go'],
    labels = ['manual'],
)
plz_e2e_test(
    name = 'num_runs_test',
    cmd = 'plz test -p --num_runs=5 //test:_num_runs_test',
    expect_output_contains = '5 passed',
)

# Tests for query affectedtests.
plz_e2e_test(
    name = 'query_affectedtests_test',
    cmd = 'plz query affectedtests -p test/affectedtests_test.go',
    expected_output = 'query_affectedtests_test.txt',
)
plz_e2e_test(
    name = 'query_affectedtests_stdin_test',
    cmd = 'echo test/affectedtests_test.go | plz query affectedtests -p -',
    expected_output = 'query_affectedtests_test.txt',
)
go_test(
    name = '_affectedtests_test',
    srcs = ['affectedtests_test.go'],
)
go_test(
    name = '_affectedtests_manual_test',
    srcs = ['affectedtests_test.go'],
    labels = ['manual'],
)

# Tests for query completions
plz_e2e_test(
    name = 'basic_completion_test',
    cmd = 'plz query completions //test/completions: | sort',
    expected_output = 'basic_completions.txt',
)
plz_e2e_test(
    name = 'build_completion_test',
    cmd = 'plz query completions //test/completions: --cmd build | sort',
    expected_output = 'basic_completions.txt',
)
plz_e2e_test(
    name = 'test_completion_test',
    cmd = 'plz query completions //test/completions: --cmd test | sort',
    expected_output = 'test_completions.txt',
)
plz_e2e_test(
    name = 'run_completion_test',
    cmd = 'plz query completions //test/completions: --cmd run | sort',
    expected_output = 'run_completions.txt',
)

# Flag tests
plz_e2e_test(
    name = 'extra_flag_test',
    cmd = 'plz cache clean',
    expected_failure = True,
)

# Test the add_out functionality which has a subtle dependency on the order
# we do things relating to the cache.
genrule(
    name = '_add_out_gen',
    cmd = 'echo hello > _add_out_gen.txt',
    post_build = lambda name, _: add_out(name, '_add_out_gen.txt'),
)
gentest(
    name = '_add_out_test',
    data = [':_add_out_gen'],
    test_cmd = 'ls test/_add_out_gen.txt',
    no_test_output = True,
    labels = ['manual'],
)
plz_e2e_test(
    name = 'add_out_test',
    cmd = 'plz build //test:_add_out_test && plz clean //test:_add_out_gen && plz test //test:_add_out_test',
)

# Test the extra output functionality.
python_test(
    name = '_extra_test_output_test',
    srcs = ['extra_test_output_test.py'],
    test_outputs = ['truth.txt'],
    labels = ['manual'],
    container = True,
    interpreter = '/usr/bin/python3',  # Docker image doesn't have /usr/bin/python
)
plz_e2e_test(
    name = 'extra_test_output_test',
    cmd = 'plz test //test:_extra_test_output_test',
    expect_file_exists = '../../../bin/test/truth.txt',
    labels = ['container', 'py3'],
)

# Test 'query alltargets'
plz_e2e_test(
    name = 'query_alltargets_1_test',
    cmd = 'plz query alltargets //test/moar/...',
    expected_output = 'query_alltargets_1.txt',
)
plz_e2e_test(
    name = 'query_alltargets_2_test',
    cmd = 'plz query alltargets //test/moar/... --include test',
    expected_output = 'query_alltargets_2.txt',
)

plz_e2e_test(
    name = 'cyclic_dependency_test',
    cmd = 'plz test //test/cycle:all',
    expect_output_contains = 'Dependency cycle found',
    expected_failure = True,
)
