subinclude("//test/build_defs")

please_repo_e2e_test(
    name = "debug_target",
    expected_output = {
        "output": "hello world",
    },
    # `--port` doesn't bring the process group where the debugging session lives is being
    # executed to the foreground, as this errors out on Circle CI.
    plz_command = "plz debug --port=31000 //:hello_world >output",
    repo = "test_repo",
)
