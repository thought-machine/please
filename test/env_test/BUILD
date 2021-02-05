subinclude("//test/build_defs")

please_repo_e2e_test(
    name = "env_test",
    expected_output = {
        "plz-out/gen/foo.txt": "foo bar",
        "plz-out/gen/bar.txt": "bar.txt foo.txt",
    },
    plz_command = "plz build //:env_test",
    repo = "test_repo",
)
