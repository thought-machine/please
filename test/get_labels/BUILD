subinclude("//test/build_defs")

def maxdepth_test(name:str, deps:list=None, maxdepth:int, expected:list):
    test_case = genrule(
        name = name,
        outs = [f"{name}.sh"],
        binary = True,
        cmd = {
            "opt": "echo '#!/bin/sh' > $OUTS",
        },
        labels = [
            f"name:{name}",
            "manual",
        ],
        output_is_complete = False,
        pre_build = echo_name_labels_up_to(maxdepth),
        deps = deps,
    )
    expected = text_file(
        name = tag(name, "expected"),
        content = "\n".join(expected) + "\n",
    )
    plz_e2e_test(
        name = f"{name}_test",
        cmd = f"plz run //test/get_labels:{name}",
        expected_output = expected,
    )

def dep(name:str, deps:list=None):
    return genrule(
        name = name,
        outs = [name],
        cmd = "touch $OUTS",
        labels = [
            f"name:{name}",
            "manual",
        ],
        output_is_complete = False,
        deps = deps,
    )

def echo_name_labels_up_to(maxdepth:int):
    def echo(name:str):
        labels = get_labels(name, "name:", maxdepth = maxdepth)
        set_command(name, "opt", " && ".join([get_command(name, "opt")] + [f"echo 'echo {l}' >> $OUTS" for l in labels]))

    return echo

dep(
    name = "dep1",
    deps = [":dep3"],
)

dep(name = "dep2")

dep(
    name = "dep3",
    deps = [
        ":dep4",
        ":dep5",
    ],
)

dep(name = "dep4")

dep(name = "dep5")

maxdepth_test(
    name = "target_only",
    expected = ["target_only"],
    maxdepth = 0,
    deps = [
        ":dep1",
        ":dep2",
    ],
)

maxdepth_test(
    name = "direct_deps",
    expected = [
        "dep1",
        "dep2",
        "direct_deps",
    ],
    maxdepth = 1,
    deps = [
        ":dep1",
        ":dep2",
    ],
)

maxdepth_test(
    name = "second_level_deps",
    expected = [
        "dep1",
        "dep2",
        "dep3",
        "second_level_deps",
    ],
    maxdepth = 2,
    deps = [
        ":dep1",
        ":dep2",
        ":dep3",
    ],
)

maxdepth_test(
    name = "all_deps",
    expected = [
        "all_deps",
        "dep1",
        "dep2",
        "dep3",
        "dep4",
        "dep5",
    ],
    maxdepth = -1,
    deps = [
        ":dep1",
        ":dep2",
    ],
)
