subinclude("//test/build_defs")

# Each package subincludes a target from the other before defining its own, so neither parse can
# ever complete. Cycle detection has to report that rather than the build hanging forever; the
# packages aren't in the graph at that point, so it can only find them via the subinclude edges.
please_repo_e2e_test(
    name = "subinclude_cycle_test",
    timeout = 60,
    expect_output_contains = {
        "output.txt": "parse of //a",
    },
    expected_failure = True,
    plz_command = "plz build //a:all //b:all > output.txt 2>&1",
    repo = "test_repo",
)
