subinclude("//test/build_defs")

please_repo_e2e_test(
    name = "plz_build_all",
    expected_failure = True,
    expected_output = {
        "plz-out/gen/package/pass": "",
        "plz-out/gen/package/dep_pass": "",
        "plz-out/gen/package/fail_test_pass_dep": "",
    },
    plz_command = "plz build --keep_going //package:all",
    repo = "test_repo",
)

please_repo_e2e_test(
    name = "plz_test_all",
    expect_output_contains = {
        "plz-out/log/test_results.xml": "fail_test_pass_dep",
    },
    expect_output_doesnt_contain = {
        "plz-out/log/test_results.xml": "fail_test_fail_dep",
    },
    expected_failure = True,
    plz_command = "plz test --keep_going //package:all",
    repo = "test_repo",
)

# Make sure we don't hang when we encounter a parse failure
# TODO(rgodden): Assert we see the outputs for //package:all once we fix the issue closing pending targets on parse
#  parse failure.
please_repo_e2e_test(
    name = "test_parse_failure",
    expected_failure = True,
    plz_command = "plz build --keep_going //package:all //package2:all",
    repo = "test_repo",
)

please_repo_e2e_test(
    name = "test_parse_empty_glob_default",
    expected_failure = True,
    plz_command = "plz build //parse_empty_glob:all",
    repo = "test_repo",
)

please_repo_e2e_test(
    name = "test_parse_empty_glob_keep_going_flag",
    expected_failure = False,
    plz_command = "plz --keep_going build //parse_empty_glob:all",
    repo = "test_repo",
)

please_repo_e2e_test(
    name = "test_parse_empty_glob_keep_going_option",
    expected_failure = False,
    plz_command = "plz -o please.keepgoing:true build //parse_empty_glob:all",
    repo = "test_repo",
)
