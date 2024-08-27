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

subinclude("//test/build_defs")

# Tests the expected output of 'query somepath'.
# Note that you have to be careful with the choice of targets, since the path
# found is not necessarily unique or stable.
plz_e2e_test(
    name = "query_somepath_test",
    cmd = "plz query somepath //src:please //src/fs",
    expected_output = "query_somepath_test.txt",
)

plz_e2e_test(
    name = "query_somepath_reverse_test",
    cmd = "plz query somepath //src/fs //src:please",
    expected_output = "query_somepath_test.txt",
)

plz_e2e_test(
    name = "query_somepath_nopath_test",
    cmd = "plz query somepath //src:please //docs/tools/config_templater",
    expected_failure = True,
    expected_output = "query_somepath_nopath_test.txt",
)

plz_e2e_test(
    name = "query_somepath_hidden_test",
    cmd = "HIDDEN=$(plz query somepath //tools/build_langserver:build_langserver //tools/build_langserver/lsp:all --hidden | wc -l); \
           NOTHIDDEN=$(plz query somepath //tools/build_langserver:build_langserver //tools/build_langserver/lsp:all | wc -l); \
           if [ $HIDDEN > $NOTHIDDEN ]; then exit 0; else exit 1; fi",
)

# Test that we count test output correctly. Also indirectly tests access to test data files.
plz_e2e_test(
    name = "test_output_test",
    cmd = "plz test //test:test_output_test_1 //test:test_output_test_2",
    expect_output_contains = "6 passed.",
)

gentest(
    name = "test_output_test_1",
    data = ["test_output_test_1.txt"],
    labels = ["manual"],
    test_cmd = "cp $(location test_output_test_1.txt) test.results",
)

gentest(
    name = "test_output_test_2",
    data = ["test_output_test_2.xml"],
    labels = ["manual"],
    test_cmd = "cp $(location test_output_test_2.xml) test.results",
)

# Test that on re-running a test it is cached.
plz_e2e_test(
    name = "test_caching_test",
    cmd = "plz test //test:caching_test && plz test -v 4 //test:caching_test",
    expect_output_contains = "Not re-running test //test:caching_test",
)

gentest(
    name = "caching_test",
    labels = ["manual"],
    no_test_output = True,
    test_cmd = "true",
    deps = ["//src:please"],
)

# Test that we don't generate coverage on running a test normally (because it's slower).
python_test(
    name = "_no_coverage_output_test",
    srcs = ["coverage_output_test.py"],
    labels = ["manual"],
)

plz_e2e_test(
    name = "no_coverage_output_test",
    cmd = "plz test //test:_no_coverage_output_test",
    expect_file_doesnt_exist = "../../../bin/test/.test_coverage__no_coverage_output_test*",
    labels = ["python3"],
)

# Test that we do generate it when using plz cover.
python_test(
    name = "_coverage_output_test",
    srcs = ["coverage_output_test.py"],
    labels = ["manual"],
)

plz_e2e_test(
    name = "coverage_output_test",
    cmd = "plz cover //test:_coverage_output_test",
    expect_file_exists = "../../../bin/test/.test_coverage__coverage_output_test*",
    # Temporarily disabled until #25 is resolved. Until then it recompiles various go_library
    # rules which makes it (and possibly others) flaky.
    labels = ["manual"],
)

# Quick test for plz run
plz_e2e_test(
    name = "plz_run_test",
    cmd = "plz run //src:please -- --version",
    expect_output_contains = "Please version",
)

# Test for query alltargets
plz_e2e_test(
    name = "query_alltargets_test",
    cmd = "plz query alltargets",
    expect_output_contains = "//src:please",
)

# Test for query output
plz_e2e_test(
    name = "query_output_test",
    cmd = "plz query output //test:query_output_filegroup",
    expected_output = "query_output_test.txt",
)

filegroup(
    name = "query_output_filegroup",
    srcs = ["//src:please"],
)

# Simulates a code generating rule to test the require / provide mechanism.
plz_e2e_test(
    name = "require_provide_test",
    cmd = "plz build //test/moar:require_provide_check -v 1 -p",
    expected_output = "require_provide_test.txt",
)

# Test for running individual tests
python_test(
    name = "individual_test_run_py",
    srcs = ["individual_test_run.py"],
    labels = ["manual"],
)

plz_e2e_test(
    name = "individual_python_test",
    cmd = "plz test //test:individual_test_run_py TestRunningIndividualTests.test_first_thing",
    expect_output_contains = "1 test target and 1 test run",
    labels = ["python3"],
)

# Test re-runs.
go_test(
    name = "num_runs_go_test",
    srcs = ["num_runs_test.go"],
    labels = ["manual"],
)

plz_e2e_test(
    name = "num_runs_test",
    cmd = "plz test -p --num_runs=5 //test:num_runs_go_test",
    expect_output_contains = "5 passed",
)

# Tests for query changes.
plz_e2e_test(
    name = "query_affected_tests_test",
    cmd = "plz query changes --level=-1 --include test test/affectedtests_test.go",
    expected_output = "query_affectedtests_test.txt",
)

plz_e2e_test(
    name = "query_affected_tests_stdin_test",
    cmd = "echo test/affectedtests_test.go | plz query changes --level=-1 --include test -",
    expected_output = "query_affectedtests_test.txt",
)

plz_e2e_test(
    name = "query_affected_tests_dependees_test",
    cmd = "plz query changes --include_dependees=transitive --include test test/affectedtests_test.go",
    expect_output_contains = "//test:affectedtests_test",
)

go_test(
    name = "affectedtests_test",
    srcs = ["affectedtests_test.go"],
)

go_test(
    name = "affectedtests_manual_test",
    srcs = ["affectedtests_test.go"],
    labels = ["manual"],
)

# Tests for query completions
plz_e2e_test(
    name = "basic_completion_test",
    cmd = "plz query completions //test/completions: | sort",
    expected_output = "basic_completions.txt",
)

plz_e2e_test(
    name = "build_completion_test",
    cmd = "plz query completions //test/completions: --cmd build | sort",
    expected_output = "basic_completions.txt",
)

plz_e2e_test(
    name = "test_completion_test",
    cmd = "plz query completions //test/completions: --cmd test | sort",
    expected_output = "test_completions.txt",
)

plz_e2e_test(
    name = "run_completion_test",
    cmd = "plz query completions //test/completions: --cmd run | sort",
    expected_output = "run_completions.txt",
)

# Flag tests
plz_e2e_test(
    name = "extra_flag_test",
    cmd = "plz cache clean",
    expected_failure = True,
)

# Test the add_out functionality which has a subtle dependency on the order
# we do things relating to the cache.
genrule(
    name = "_add_out_gen",
    cmd = "echo hello > _add_out_gen.txt",
    post_build = lambda name, _: add_out(name, "_add_out_gen.txt"),
)

gentest(
    name = "_add_out_test",
    data = [":_add_out_gen"],
    labels = ["manual"],
    no_test_output = True,
    test_cmd = "ls test/_add_out_gen.txt",
)

plz_e2e_test(
    name = "add_out_test",
    cmd = "plz build //test:_add_out_test && plz clean //test:_add_out_gen && plz test //test:_add_out_test",
)

# Test the extra output functionality.
go_test(
    name = "extra_test_output_go_test",
    srcs = ["extra_test_output_test.go"],
    labels = ["manual"],
    test_outputs = ["truth.txt"],
)

plz_e2e_test(
    name = "extra_test_output_test",
    cmd = "plz test //test:extra_test_output_go_test",
    expect_file_exists = "../../../../bin/test/truth.txt",
)

# Test 'query alltargets'
plz_e2e_test(
    name = "query_alltargets_1_test",
    cmd = "plz query alltargets //test/moar/...",
    expected_output = "query_alltargets_1.txt",
)

plz_e2e_test(
    name = "query_alltargets_2_test",
    cmd = "plz query alltargets //test/moar/... --include test",
    expected_output = "query_alltargets_2.txt",
)

plz_e2e_test(
    name = "cyclic_dependency_test",
    timeout = 30,
    cmd = "plz test //plz-out/tmp/test/cyclic_dependency_test._test/run_1/test/cycle:all",
    data = ["cycle/TEST_BUILD"],
    expect_output_contains = "Dependency cycle found",
    expected_failure = True,
    pre_cmd = "mv test/cycle/TEST_BUILD test/cycle/BUILD",
)

# Used manually for testing the test flakiness stuff.
python_test(
    name = "flaky_test",
    srcs = ["flaky_test.py"],
    flaky = True,
    labels = ["manual"],
)

# Tests on commands / flags etc.
plz_e2e_test(
    name = "unknown_command_test",
    cmd = "plz fix",
    expect_output_contains = "Unknown command",
    expected_failure = True,
)

plz_e2e_test(
    name = "unknown_flag_test",
    cmd = "plz build --wibble",
    expect_output_contains = "unknown flag",
    expected_failure = True,
)

# Tests on the stamp attribute.
# These essentially pass as long as they can build.
# (Note that we can't use -v because OSX is unlikely to have a new enough version of bash).
build_rule(
    name = "stamp_negative_test",
    cmd = "[ -z ${STAMP+x} ]",
    no_test_output = True,
    test = True,
    test_cmd = "true",
)

build_rule(
    name = "stamp_positive_test",
    cmd = "[ ! -z ${STAMP+x} ]",
    no_test_output = True,
    stamp = True,
    test = True,
    test_cmd = "true",
)

# Test on a build rule that writes a symlink.
genrule(
    name = "symlink_gen",
    srcs = ["symlink_test.txt"],
    outs = ["symlink_test.txt"],
    cmd = 'ln -s $SRCS "$OUTS"',
)

gentest(
    name = "symlink_test",
    data = [":symlink_gen"],
    no_test_output = True,
    test_cmd = "test -L test/symlink_test.txt",
)

# This rule tests the no_test_output flag. If that isn't honoured
# plz would report a test failure because results were missing.
gentest(
    name = "no_test_output_test",
    no_test_output = True,
    test_cmd = "echo SUCCESS",
)

# This tests that data files exist in the correct location, and
# indirectly performs a basic test of sh_test which we don't use elsewhere.
sh_test(
    name = "data_files_test",
    src = "data_files_test.sh",
    data = ["container_data.txt"],
)

# This test is here as a convenience to test the flakiness functionality.
# It's just using random internally so won't pass consistently.
python_test(
    name = "flakiness_test",
    srcs = ["flakiness_test.py"],
    flaky = 5,
    labels = ["manual"],
)

go_binary(
    name = "test",
    srcs = ["name_conflict.go"],
)

genrule(
    name = "pass_env",
    outs = ["pass_env.sh"],
    cmd = "echo \"export USER='$USER'\" > \"$OUT\"",
    pass_env = ["USER"],
)

sh_test(
    name = "pass_env_test",
    src = "pass_env_test.sh",
    data = [":pass_env"],
    labels = ["manual"],  # Unclear what a good env var to pass is that works locally & on CI.
)

genrule(
    name = "print_rule",
    cmd = "true",
    labels = [
        "a:hello",
        "a:greetings",
        "b:goodbye",
        "hello",
        "manual",
    ],
)

plz_e2e_test(
    name = "query_print_test",
    cmd = "plz query print -l a: //test:print_rule",
    expected_output = "query_print_test.txt",
)

gentest(
    name = "failing_test",
    labels = ["manual"],
    test_cmd = "false",
)

plz_e2e_test(
    name = "failing_tests_ok_test",
    cmd = "plz test //test:failing_test --failing_tests_ok",
)

plz_e2e_test(
    name = "init_test",
    cmd = "plz init --dir=$TEST_DIR --no_prompt",
    expect_file_exists = ".plzconfig",
)

plz_e2e_test(
    name = "completion_test",
    cmd = "plz --completion_script",
    expect_output_contains = "plz completion",
)

# This rule being here makes sure than when we parse this file, none of the test fields being set causes a nil pointer
# even though this target isn't a test.
build_rule(
    name = "non_test_target_with_test_fields_set",
    flaky = 3,
    labels = ["manual"],
    no_test_output = True,
    test = False,
    test_cmd = "true",
    test_outputs = ["foo"],
    test_sandbox = True,
    test_timeout = 60,
    test_tools = ["go"],
)

plz_e2e_test(
    name = "non_test_target_with_test_fields_set_test",
    cmd = "plz test //test:non_test_target_with_test_fields_set",
)

gentest(
    name = "shell_output_test_case",
    labels = ["manual"],
    no_test_output = True,
    test_cmd = "echo '$TEST'; exit 1",
)

plz_e2e_test(
    name = "shell_output_test",
    cmd = "plz test //test:shell_output_test_case",
    expect_output_contains = "$TEST",
    expected_failure = True,
)

gentest(
    name = "rm_ro_test_case",
    labels = ["manual"],
    no_test_output = True,
    test_cmd = " && ".join([
        "mkdir subdir",
        "touch subdir/file.txt",
        "chmod -w subdir/file.txt",
        "chmod -w subdir",
    ]),
)

plz_e2e_test(
    name = "rm_ro_test",
    cmd = "plz test --rerun //test:rm_ro_test_case",
    expect_output_doesnt_contain = "ERROR",
)
