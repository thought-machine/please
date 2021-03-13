subinclude("//test/build_defs")

please_repo_e2e_test(
    name = "entry_point_test",
    expected_output = {
        "plz-out/gen/foo/wibble.txt": "wibble",
    },
    plz_command = "plz build //foo",
    repo = "test_repo",
)

please_repo_e2e_test(
    name = "run_entry_point_arg_test",
    expected_output = {
        "wibble.txt": "wibble",
    },
    plz_command = "plz run -e printer //tools:my_tool > wibble.txt",
    repo = "test_repo",
)

please_repo_e2e_test(
    name = "run_entry_point_annotation_test",
    expected_output = {
        "wibble.txt": "wibble",
    },
    plz_command = 'plz run "//tools:my_tool|printer" > wibble.txt',
    repo = "test_repo",
)

please_repo_e2e_test(
    name = "command_expansion_test",
    expected_output = {
        "wibble.txt": "wibble",
    },
    plz_command = "plz run //foo:foo_cmd",
    repo = "test_repo",
)
