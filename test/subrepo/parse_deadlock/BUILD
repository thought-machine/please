subinclude("//test/build_defs")

please_repo_e2e_test(
    name = "parse_deadlock_test",
    expected_output = {
        "plz-out/gen/pkg/x.txt": "hello",
    },
    # Bounded so a regression fails the test rather than hanging until the CI job is killed.
    plz_command = "timeout 120 plz build //pkg:x",
    repo = "test_repo",
)
