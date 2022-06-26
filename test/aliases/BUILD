subinclude("//test/build_defs")
please_repo_e2e_test(
    name = "flat_alias_config_test",
    expected_output = {
        "output": "wow 2 args",
    },
    plz_command = "plz oldschool --flag arg >output",
    repo = "test_repo",
)

please_repo_e2e_test(
    name = "new_aliasconfig_test",
    expected_output = {
        "output": "wow 1 args",
    },
    plz_command = "plz meme --nocreate >output",
    repo = "test_repo",
)

please_repo_e2e_test(
    name = "new_aliasconfig_with_nested_subfile_test",
    expected_output = {
        "output": "astonished 2",
    },
    plz_command = "plz dict deen --ffs arg >output",
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
    name = "alias_config_choices_completion_test",
    expect_output_contains = {
        "output": "big",
    },
    plz_command = "export GO_FLAGS_COMPLETION=1; plz meme --doge b >output",
    repo = "test_repo",
)

please_repo_e2e_test(
    name = "alias_config_choices_test",
    expected_output = {
        "output": "wow 1 args",
    },
    plz_command = "plz meme --doge=big >output",
    repo = "test_repo",
)

please_repo_e2e_test(
    name = "alias_config_help_options_test",
    expect_output_contains = {
        "out.txt": "Usage:\n  meme\n\nGenerates a meme.\n  --doge               Specifies a doge meme.\n  --nocreate           Whether to create the directory if it does not exist.\n",
    },
    plz_command = "plz meme --help > out.txt",
    repo = "test_repo",
)
