subinclude("//test/build_defs")

please_repo_e2e_test(
    name = "test_subrepo",
    expected_output = {
        "out.txt": "//subrepo:gotest\n//subrepo/dir:gotest\n//subrepo/dir:pleasings2",
    },
    plz_command = "plz --profile local query alltargets //subrepo/... > out.txt",
    repo = "test_repo",
)

please_repo_e2e_test(
    name = "test_local",
    expected_output = {
        "plz-out/gen/local/out": "constant",
        "plz-out/gen/local/foo/out": "constant",
    },
    plz_command = "plz build --profile local //local/...",
    repo = "test_repo",
)