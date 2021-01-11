subinclude("//test/build_defs")

please_repo_e2e_test(
    name = "test",
    expected_output = {
        "out.txt": "//:gotest\n//:pleasings\n//dir:gotest\n//dir:pleasings2",
    },
    plz_command = "plz query alltargets > out.txt",
    repo = "test_repo",
)
