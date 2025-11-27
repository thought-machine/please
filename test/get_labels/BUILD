subinclude("//test/build_defs")

def maxdepth_test(name:str, deps:list=None, maxdepth:int, expected:list):
    test_case = genrule(
        name = name,
        outs = [f"{name}.sh"],
        cmd = {
            "opt": "echo '#!/bin/sh' > $OUTS",
        },
        binary = True,
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
        labels = get_labels(name, "name:", transitive=True, maxdepth=maxdepth)
        set_command(name, "opt", " && ".join([get_command(name, "opt")] + [f"echo 'echo {l}' >> $OUTS" for l in labels]))
    return echo

dep(name = "dep1", deps = [":dep3"])
dep(name = "dep2")
dep(name = "dep3", deps = [":dep4", ":dep5"])
dep(name = "dep4")
dep(name = "dep5")

maxdepth_test(
    name = "target_only",
    deps = [
        ":dep1",
        ":dep2",
    ],
    maxdepth = 0,
    expected = ["target_only"],
)

maxdepth_test(
    name = "direct_deps",
    deps = [
        ":dep1",
        ":dep2",
    ],
    maxdepth = 1,
    expected = [
        "dep1",
        "dep2",
        "direct_deps",
    ],
)

maxdepth_test(
    name = "second_level_deps",
    deps = [
        ":dep1",
        ":dep2",
    ],
    maxdepth = 2,
    expected = [
        "dep1",
        "dep2",
        "dep3",
        "second_level_deps",
    ],
)

maxdepth_test(
    name = "all_deps",
    deps = [
        ":dep1",
        ":dep2",
    ],
    maxdepth = -1,
    expected = [
        "all_deps",
        "dep1",
        "dep2",
        "dep3",
        "dep4",
        "dep5",
    ],
)
