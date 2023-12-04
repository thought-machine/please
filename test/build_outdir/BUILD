subinclude("//test/build_defs")

please_repo_e2e_test(
    name = "out_dir_test",
    expected_output = {"bar/foo/testy": "wibble wibble wibble"},
    plz_command = "plz build //foo:testy --out_dir=bar",
    repo = "test_repo",
)
