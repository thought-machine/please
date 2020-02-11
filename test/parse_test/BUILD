# Simulates a code generating rule to test the require / provide mechanism.
genrule(
    name = "test_require_py",
    outs = ["test_require.py"],
    cmd = 'touch "$OUT"',
    test_only = True,
)

genrule(
    name = "test_require_go",
    outs = ["test_require.go"],
    cmd = 'touch "$OUT"',
    test_only = True,
)

filegroup(
    name = "test_require",
    srcs = [
        ":test_require_go",
        ":test_require_py",
    ],
    provides = {
        "py": ":test_require_py",
        "go": ":test_require_go",
    },
    test_only = True,
    deps = [
        ":test_require_go",
        ":test_require_py",
    ],
)

python_test(
    name = "require_provide_test",
    srcs = ["require_provide_test.py"],
    deps = [
        ":test_require",
    ],
)

# Test for adding additional outputs to a target.
genrule(
    name = "_gen_output_name",
    cmd = "echo test_additional_output.txt",
    post_build = lambda _, output: add_out(
        "_gen_output",
        "".join(output).strip(),
    ),
)

genrule(
    name = "_gen_output",
    cmd = 'echo -n "kittens" > "$OUT"',
    deps = [
        ":_gen_output_name",
    ],
)

go_test(
    name = "additional_output_test",
    srcs = ["additional_output_test.go"],
    data = [":_gen_output"],
)

# Test for a target with a missing tool; we should still be able to parse the package.
genrule(
    name = "missing_tool",
    cmd = "$TOOL --version",
    labels = ["manual"],
    tools = ["python4.7"],
)
