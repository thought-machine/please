subinclude("//test/build_defs")

filegroup(
    name = "target1",
    deps = [":target4"],
)

filegroup(
    name = "target2",
    labels = ["branch"],
    deps = [
        ":target1",
        ":target4",
    ],
)

filegroup(
    name = "target3",
    labels = ["root"],
    deps = [
        ":target2",
    ],
)

filegroup(
    name = "target4",
    labels = ["leaf"],
)

plz_e2e_test(
    name = "query_revdeps_branch_test",
    cmd = "plz query revdeps --level=-1 //test/revdeps:target1 --include branch",
    expected_output = "query_revdeps_branch_test.txt",
)

plz_e2e_test(
    name = "query_revdeps_root_test",
    cmd = "plz query revdeps --level=-1 //test/revdeps:target1 --include root",
    expected_output = "query_revdeps_root_test.txt",
)

plz_e2e_test(
    name = "query_revdeps_single_level_test",
    cmd = "plz query revdeps --level=1 //test/revdeps:target1",
    expected_output = "query_revdeps_single_level_test.txt",
)

plz_e2e_test(
    name = "query_revdeps_no_level_test",
    cmd = "plz query revdeps --level=0 //test/revdeps:target1",
    expected_output = "query_revdeps_no_level_test.txt",
)

# Test that we always include 3 when querying deps at level 2
# 3 -> 2 -> 1 -> 4 (3 is 3 levels deep here so would be excluded)
# 3 -> 2 -> 4 (but is at level 2 here so should be included)
plz_e2e_test(
    name = "query_revdeps_multiple_paths_test",
    cmd = "plz query revdeps --level=2 //test/revdeps:target4",
    expected_output = "query_revdeps_multiple_paths_test.txt",
)
