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
