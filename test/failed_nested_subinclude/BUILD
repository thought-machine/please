subinclude("//test/build_defs")

# Both packages subinclude the same file, which in turn subincludes a target that fails to build.
# Only one of them interprets that file; the other waits for it, and must see the failure rather
# than hang waiting for a result that never gets recorded.
please_repo_e2e_test(
    name = "failed_nested_subinclude_test",
    timeout = 60,
    expect_output_contains = {
        "output": "Error building target //defs:slow",
    },
    expected_failure = True,
    plz_command = "plz build //a:x //b:x > output 2>&1",
    repo = "test_repo",
)
