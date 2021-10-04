subinclude("//test/build_defs")

please_repo_e2e_test(
    name = "debug_target",
    expected_output = {
        "output": "hello world\nlorem ipsum...",
    },
    # `--port` is used here to prevent this test from erroring out on Circle CI,
    # where it won't bring the process group where the debugging session is being
    # executed to the foreground.
    plz_command = "plz debug --port=31000 //:hello_world >output",
    repo = "test_repo",
)
