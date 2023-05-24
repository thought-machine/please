subinclude("//test/build_defs")

text_file(
    name = "file",
    content = "wibble",
)

text_file(
    name = "multilines_file",
    strip = True,
    content = """
        This is a multi lines file,
        use strip option to remove first empty newline
        and all empty spaces on the left of the text.
    """,
)

text_file(
    name = "expand_file",
    content = "$(location :file)",
    deps = [":file"],
)

gentest(
    name = "file_test",
    data = [":file"],
    labels = ["rex"],
    no_test_output = True,
    test_cmd = "$TOOL $(location :file) wibble",
    test_tools = ["//test/build_defs:content_checker"],
)

gentest(
    name = "multilines_file_test",
    data = [":multilines_file"],
    labels = ["rex"],
    no_test_output = True,
    test_cmd = """
        $TOOL $(location :multilines_file) \
            'This is a multi lines file,' \
            'use strip option to remove first empty newline' \
            'and all empty spaces on the left of the text.'
    """,
    test_tools = ["//test/build_defs:content_checker"],
)

gentest(
    name = "expand_file_test",
    data = [":expand_file"],
    labels = ["rex"],
    no_test_output = True,
    test_cmd = "$TOOL $(location :expand_file) " + package_name() + "/file",
    test_tools = ["//test/build_defs:content_checker"],
)
