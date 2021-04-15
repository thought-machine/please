subinclude("//test/build_defs")

please_repo_e2e_test(
    name = "optional_outs_test",
    expected_output = {
        "plz-out/gen/foo/foo.sym": "",
        "plz-out/gen/foo/foo.wildcard.sym": "",
    },
    plz_command = "plz build //foo && rm plz-out/gen/foo/* && plz build //foo",
    repo = "test_repo",
)
