subinclude("//test/build_defs")
# the args are there but aren't appearing
please_repo_e2e_test(
    name = "flat_alias_config_test",
    expected_output = {
        "output": "wow 1 args",
    },
    plz_command = "plz oldschool --flag arg >output",
    repo = "test_repo",
)

please_repo_e2e_test(
    name = "alias_config_test",
    expected_output = {
        "output": "wow 0 args",
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
    name = "alias_config_help_options_test",
    expect_output_contains = {
        "out.txt": "Usage:\n  meme\n\nGenerates a meme.\n  --doge               Specifies a doge meme.\n  --nocreate           Whether to create the directory if it does not exist.\n",
    },
    plz_command = "plz meme --help > out.txt",
    repo = "test_repo",
)

# the args aren't appearing for this one - change back to 1 to test
please_repo_e2e_test(
    name = "alias_config_subfile_test",
    expected_output = {
        "output": "astonished 1",
    },
    plz_command = "plz dict deen --ffs >output",
    repo = "test_repo",
)