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

