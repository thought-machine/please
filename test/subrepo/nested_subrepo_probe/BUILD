subinclude("//test/build_defs")

# Resolving a subrepo that is declared inside another subrepo makes Please parse the package it
# would be declared in in the host repo first, to see if it is there. When it isn't - which is the
# normal case, not an error - that lookup used to claim the right to parse the package and never
# give it back, so the next thing to ask about the same package waited on a parse nobody was going
# to do. The build hung with no output at all rather than failing.
#
# Two subrepos are needed to see it: the first one to look poisons the lookup, the second one
# waits on it. timeout is what makes this a failure rather than a hang.
please_repo_e2e_test(
    name = "nested_subrepo_probe_test",
    expected_output = {
        "plz-out/gen/suba/a.txt": "one",
        "plz-out/gen/subb/b.txt": "two",
    },
    plz_command = "timeout 180 plz build //:both",
    repo = "test_repo",
)
