subinclude("//test/build_defs")

please_repo_e2e_test(
    name = "alias_config_test",
    expected_output = {
        "output": "wow",
    },
    plz_command = "plz meme >output",
    repo = "test_repo",
)

please_repo_e2e_test(
    name = "alias_config_completion_test",
    expect_output_contains = {
        "output": "doge",
    },
    plz_command = "export GO_FLAGS_COMPLETION=1; plz meme --dog >output",
    repo = "test_repo",
)

please_repo_e2e_test(
    name = "alias_config_help_test",
    expect_output_contains = {
        "output": "Generates a meme.",
    },
    plz_command = "plz meme --help >output",
    repo = "test_repo",
)
