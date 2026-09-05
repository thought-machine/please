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

# Regression test: a debug tool that provide()s something its dependent require()s must still be
# built as itself, otherwise it's missing at test time under -c dbg (as happens with the Go plugin's
# delve_tool, where every go_binary provides a "go" target and every go_test requires "go").
please_repo_e2e_test(
    name = "debug_tool_require_provide",
    plz_command = "plz test -c dbg //:debug_tool_test && test -f plz-out/bin/debugger.sh",
    repo = "test_repo",
)
