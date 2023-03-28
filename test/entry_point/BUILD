subinclude("//test/build_defs")

def e2e_test(name, expected_output, plz_command):
    please_repo_e2e_test(
        name = name,
        expected_output = expected_output,
        plz_command = plz_command.replace("plz ", "plz -o plugin.go.gotool:$TOOLS_GO "),
        tools = {
            "go": [CONFIG.GO.GO_TOOL],
        },
        repo = "test_repo",
    )

e2e_test(
    name = "entry_point_test",
    expected_output = {
        "plz-out/gen/foo/wibble.txt": "wibble",
    },
    plz_command = "plz build //foo",
)

e2e_test(
    name = "run_entry_point_arg_test",
    expected_output = {
        "wibble.txt": "wibble",
    },
    plz_command = "plz run -e printer //tools:my_tool > wibble.txt",
)

e2e_test(
    name = "run_entry_point_annotation_test",
    expected_output = {
        "wibble.txt": "wibble",
    },
    plz_command = 'plz run "//tools:my_tool|printer" > wibble.txt',
)

e2e_test(
    name = "run_entry_point_annotation_tmp_dir_test",
    expected_output = {
        "wibble.txt": "wibble",
    },
    plz_command = 'plz run --in_tmp_dir "//tools:my_tool|printer" > wibble.txt',
)

e2e_test(
    name = "command_expansion_test",
    expected_output = {
        "wibble.txt": "wibble",
    },
    plz_command = "plz run //foo:foo_cmd",
)
