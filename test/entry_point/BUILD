subinclude("//test/build_defs")

please_repo_e2e_test(
    name = "entry_point_test",
    expected_output = {
        "plz-out/gen/foo/wibble.txt": "wibble",
    },
    plz_command = "plz build //foo",
    repo = "test_repo",
)
