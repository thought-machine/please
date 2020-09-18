subinclude("//test/build_defs")

please_repo_e2e_test(
    name = "go_test_compat_test",
    data = {
        "TEST_TOOL": ["//tools/please_go_test"],
        "FILTER_TOOL": ["//tools/please_go_filter"],
    },
    expected_output = {
        "plz-out/bin/foo/bar.txt": "wibble wibble wibble",
    },
    plz_command = "$PLZ test -o go.testtool:$TMP_DIR/$DATA_TEST_TOOL -o go.filtertool:$TMP_DIR/$DATA_FILTER_TOOL //foo/...",
    repo = "test_repo",
)
