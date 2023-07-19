subinclude("//test/build_defs")

def provider(name, deps=[]):
    # This is the rule that actually gets depended on after resolving the require/provide setup
    go = build_rule(
        name = name,
        outs = [name],
        cmd = "touch $OUT",
        requires = ["go"],
        tag = "go",
        deps = deps,
    )

    return filegroup(
        name = name,
        provides = {
            "go": go,
        },
        deps = [go],
    )

provider(
    name = "one",
)

provider(
    name = "two",
    deps = [":one"],
)

provider(
    name = "three",
    deps = [":two"],
)

plz_e2e_test(
    name = "require_provides_revdeps_test",
    cmd = "plz query revdeps --level=-1 //test/revdeps/require_provide:one",
    expected_output = "revdeps.txt",
)

plz_e2e_test(
    name = "require_provides_some_path_test",
    cmd = "plz query somepath //test/revdeps/require_provide:three //test/revdeps/require_provide:two",
    expected_output = "somepath.txt",
)
