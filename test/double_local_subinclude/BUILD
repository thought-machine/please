subinclude("//test/build_defs")

please_repo_e2e_test(
    name = "double_local_subinclude_test",
    plz_command = "plz build -p -v 2 //...",
    expected_failure = True,
    repo = "test_repo",
)
