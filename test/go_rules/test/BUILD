# For testing the go build commands, where we need to be careful about things
# being in subdirectories.
go_library(
    name = "test",
    srcs = ["test.go"],
    test_only = True,
    visibility = ["//test/..."],
)

# Tests a go 'external test', where you can have files in the same directory
# with a _test suffix.
go_test(
    name = "external_test",
    srcs = ["external_test.go"],
    definitions = {
        "test/go_rules/test.Var": "var",
        "test/go_rules/test.Var2": "var1 var2",

        "test/go_rules/test.ExecList": exec(
            cmd = ["echo", "stdout1;", "echo", "1>&2", "stderr;", "echo", "   stdout2   "], stdout = True, stderr = False, cache = True),
        "test/go_rules/test.ExecStr": exec(
            cmd = "echo stdout1; echo 1>&2 stderr; echo    stdout2   ", cache = False),
        "test/go_rules/test.ExecStdErr": exec(
            cmd = ["sh", "-c", "echo stdout1; echo 1>&2 stderr; echo stdout2"],
            stdout = False, stderr = True),
        "test/go_rules/test.ExecCombinedOut": exec(
            cmd = ["sh", "-c", "echo stdout1; echo 1>&2 stderr; echo stdout2"],
            stdout = True, stderr = True),
    },
    external = True,
    deps = [
        ":test",
        "//third_party/go:testify",
    ],
)
