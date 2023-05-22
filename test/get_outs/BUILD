nonamedouts = genrule(
    name = "nonamedouts",
    outs = ["x"],
    cmd = """
        echo 'x' > "$OUTS"
    """,
)

gr_getouts = genrule(
    name = "genrule_getouts",
    outs = {
        "wibble": ["wibble_file1"],
        "wobble": ["wobble_file1"],
    },
    cmd = """
        echo 'wibblewibblewibble' > "$OUTS_WIBBLE"
        echo 'wobblewobblewobble' > "$OUTS_WOBBLE"
    """,
)

fg_getouts = filegroup(
    name = "filegroup_getouts",
    srcs = {
        "wibble": [text_file(name = 'wibble_file2', content = 'wibblewibblewibble')],
        "wobble": [text_file(name = 'wobble_file2', content = 'wobblewobblewobble')],
    },
)

def assert_dict(l1, l2):
    if l1 != l2:
        fail(f"{l1} != {l2}")

assert_dict({}, get_named_outs(nonamedouts))

assert_dict({
    "wibble": ["//test/get_outs:genrule_getouts|wibble"],
    "wobble": ["//test/get_outs:genrule_getouts|wobble"],
}, get_named_outs(gr_getouts))

assert_dict({
    "wibble": ["//test/get_outs:filegroup_getouts|wibble"],
    "wobble": ["//test/get_outs:filegroup_getouts|wobble"],
}, get_named_outs(fg_getouts))

gr_subtargets = get_named_outs(gr_getouts)
fg_subtargets = get_named_outs(fg_getouts)

gentest(
    name = "get_outs_gr_wibble_test",
    data = gr_subtargets.wibble,
    labels = ["get_outs"],
    no_test_output = True,
    test_cmd = """
        $TOOL "$DATA" "wibblewibblewibble"
    """,
    test_tools = ["//test/build_defs:content_checker"],
)

gentest(
    name = "get_outs_gr_wobble_test",
    data = gr_subtargets.wobble,
    labels = ["get_outs"],
    no_test_output = True,
    test_cmd = """
        $TOOL "$DATA" "wobblewobblewobble"
    """,
    test_tools = ["//test/build_defs:content_checker"],
)

gentest(
    name = "get_outs_fg_wibble_test",
    data = fg_subtargets.wibble,
    labels = ["get_outs"],
    no_test_output = True,
    test_cmd = """
        $TOOL "$DATA" "wibblewibblewibble"
    """,
    test_tools = ["//test/build_defs:content_checker"],
)

gentest(
    name = "get_outs_fg_wobble_test",
    data = fg_subtargets.wobble,
    labels = ["get_outs"],
    no_test_output = True,
    test_cmd = """
        $TOOL "$DATA" "wobblewobblewobble"
    """,
    test_tools = ["//test/build_defs:content_checker"],
)
