subinclude("//test/build_defs")

please_repo_e2e_test(
    name = "go_test_compat_test",
    expected_output = {
        "plz-out/bin/foo/bar.txt": "wibble wibble wibble",
    },
    plz_command = "plz test //foo/...",
    repo = "test_repo",
)
