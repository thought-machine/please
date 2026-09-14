subinclude("//test/build_defs")

please_repo_e2e_test(
    name = "subinclude_subrepo_deadlock_test",
    # Regression test for a deadlock between parsing a package and evaluating a subincluded file that
    # reaches into a subrepo. subinclude() of a label in a subrepo waits for the *whole* package that
    # defines that subrepo (//third_party:all) before it does anything else, rather than for the
    # subrepo itself, so:
    #   - the goroutine parsing //third_party owns that package's parse claim, and on its own
    #     subinclude("//rules:defs") line takes the claim on rules/defs.build_defs;
    #   - evaluating rules/defs.build_defs then waits for //third_party:all - a claim held either by
    #     that same goroutine, or by it while the //app goroutine waits on rules/defs.build_defs.
    # Nothing here is circular: third_party registers the subrepo before its own subinclude, so
    # everything rules/defs.build_defs needs is available well before anything waits for it - drop
    # third_party's subinclude() line and the same repo builds fine. Only the ':all' coarsening makes
    # it look like a cycle, and since no BuildTarget depends on another the graph cycle detector
    # can't see it either.
    # Bounded so a regression fails the test rather than hanging until the CI job is killed.
    timeout = 60,
    expected_output = {
        "plz-out/gen/app/a.txt": "hello",
        "plz-out/gen/third_party/z.txt": "hello",
    },
    plz_command = "plz build //app:a //third_party:z",
    repo = "test_repo",
)
