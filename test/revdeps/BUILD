subinclude("//build_defs:plz_e2e_test")

filegroup(
    name = "target1",
    labels = ["leaf"],
)

filegroup(
    name = "target2",
    labels = ["branch"],
    deps = [":target1"],
)

filegroup(
    name = "target3",
    labels = ["root"],
    deps = [":target2"],
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
