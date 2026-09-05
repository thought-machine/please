subinclude("//test/build_defs")

please_repo_e2e_test(
    name = "wildcard_test",
    expected_output = {
        "plz-out/gen/subrepo/root": "root",
        "plz-out/gen/subrepo/pkg/nested": "nested",
    },
    plz_command = "plz build ///subrepo//...",
    repo = "test_repo",
)
