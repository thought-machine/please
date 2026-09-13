subinclude("//test/build_defs")

please_repo_e2e_test(
    name = "subinclude_race_test",
    expected_output = {
        "plz-out/gen/a/x.txt": "hello",
        "plz-out/gen/b/y.txt": "hello",
    },
    plz_command = "plz build //a:x //b:y",
    repo = "test_repo",
    # Regression test for a deadlock between two packages that each subinclude their own output, where
    # that cascades into needing a target defined by the other package (a:self -> b:self -> b:dep ->
    # a:dep). Depending on which package's parse gets claimed first, the goroutine parsing one can end
    # up waiting for the other package's whole parse to finish while that parse is itself waiting on it.
    # It's not a cycle in the build graph (no BuildTarget depends on the other) nor a subinclude content
    # cycle (the chain of files subincluded is a DAG, never revisiting a path), so neither of those is
    # able to catch it - only requesting both packages independently, in parallel, reproduces it.
    # Bounded so a regression fails the test rather than hanging until the CI job is killed.
    timeout = 60,
)
