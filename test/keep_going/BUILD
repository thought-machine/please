subinclude("//test/build_defs")

please_repo_e2e_test(
    name = "build_all",
    expected_failure = True,
    plz_command = "plz build --keep_going //package:all",
    repo = "test_repo",
)

