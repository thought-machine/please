subinclude("//test/build_defs")

please_repo_e2e_test(
    name = "same_package_error",
    expect_output_contains = {
        "output": "is not defined in this package yet",
    },
    expected_failure = True,
    plz_command = "plz build //:all 2>output",
    repo = "test_repo",
)
