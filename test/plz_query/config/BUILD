subinclude("//test/build_defs")

plz_e2e_test(
    name = "config",
    cmd = "plz query config",
    expect_output_contains = """
buildfilename = BUILD
buildfilename = BUILD.plz
""",
)

plz_e2e_test(
    name = "config_options",
    cmd = "plz query config parse.buildfilename size.small.timeout",
    expect_output_contains = """BUILD
BUILD.plz
60000000000""",
)

please_repo_e2e_test(
    name = "plugin_config_field_default_value",
    repo = "test_repo",
    data = {"plugin": [":foo_plugin"]},
    plz_command = "plz -o please.pluginrepo:file://$TMP_DIR/$DATA_PLUGIN query config plugin.foo.fooctool plugin.foo.bartools > output.txt",
    expected_output = {"output.txt": """fooc
///foo//tools/bar:bar
///baz//tools/bar:bar
"""},
)

genrule(
    name = "foo_plugin",
    srcs = {
        "plugin": ["foo_plugin"],
        "tool": ["//test/plugins/tools:fooc"],
    },
    outs = ["foo_plugin.tar.gz"],
    cmd = "mv $SRCS_PLUGIN foo_plugin && mv $SRCS_TOOL foo_plugin/tools && tar -czf $OUT foo_plugin",
)

plz_e2e_test(
    name = "json_config",
    cmd = "plz query config --json",
    expect_output_contains = """
        \\"buildfilename\\": [
            \\"BUILD\\",
            \\"BUILD.plz\\"
        \\],
    """,
)

plz_e2e_test(
    name = "json_and_options",
    cmd = "plz query config --json parse.buildfilename",
    expected_failure = True,
)

plz_e2e_test(
    name = "invalid_option",
    cmd = "plz query config fake_option",
    expected_failure = True,
)
