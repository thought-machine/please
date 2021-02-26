subinclude("//test/build_defs")

text_file(
    name = "file",
    content = "wibble",
)

gentest(
    name = "file_test",
    data = [":file"],
    labels = ["rex"],
    no_test_output = True,
    test_cmd = "$TOOL $(location :file) wibble",
    test_tools = ["//test/build_defs:content_checker"],
)
