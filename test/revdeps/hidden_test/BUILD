subinclude("//test/build_defs")

# creates a filegroup with two hidden intermediate rules
def two_intermediate(name, deps):
    one = filegroup(
        name = name,
        tag = "one",
        deps = deps,
    )

    two = filegroup(
        name = name,
        tag = "two",
        deps = [one],
    )

    return filegroup(
        name = name,
        deps = [two],
    )

two_intermediate(
    name = "lvl0",
    deps = [],
)

two_intermediate(
    name = "lvl1",
    deps = [":lvl0"],
)

two_intermediate(
    name = "lvl2",
    deps = [":lvl1"],
)

two_intermediate(
    name = "lvl3",
    deps = [":lvl2"],
)

two_intermediate(
    name = "lvl4",
    deps = [":lvl3"],
)

plz_e2e_test(
    name = "query_revdeps_level1",
    cmd = "plz query revdeps --level=1 //test/revdeps/hidden_test:lvl0",
    expected_output = "lvl1.txt",
)

plz_e2e_test(
    name = "query_revdeps_level2",
    cmd = "plz query revdeps --level=2 //test/revdeps/hidden_test:lvl0",
    expected_output = "lvl2.txt",
)

plz_e2e_test(
    name = "query_revdeps_level3",
    cmd = "plz query revdeps --level=3 //test/revdeps/hidden_test:lvl0",
    expected_output = "lvl3.txt",
)

plz_e2e_test(
    name = "query_revdeps_level4",
    cmd = "plz query revdeps --level=4 //test/revdeps/hidden_test:lvl0",
    expected_output = "lvl4.txt",
)

plz_e2e_test(
    name = "query_revdeps_level4_hidden",
    cmd = "plz query revdeps --level=4 --hidden //test/revdeps/hidden_test:lvl0",
    expected_output = "lvl4_hidden.txt",
)
