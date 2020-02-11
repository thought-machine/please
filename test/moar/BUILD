# For testing certain things that require targets to be in a separate package.

package(default_visibility = ["//test/..."])

genrule(
    name = "test_require_py",
    outs = ["test_require.py"],
    cmd = 'touch "$OUT"',
)

genrule(
    name = "test_require_go",
    outs = ["test_require.go"],
    cmd = 'touch "$OUT"',
)

filegroup(
    name = "test_require_fg",
    srcs = [
        ":test_require_go",
        ":test_require_py",
    ],
    provides = {
        "py": ":test_require_py",
        "go": ":test_require_go",
    },
    deps = [
        ":test_require_go",
        ":test_require_py",
    ],
)

python_library(
    name = "require_provide_check",
    srcs = [],
    deps = [
        ":test_require_fg",
    ],
)

# Test that require/provide updates $SRCS correctly.
# See issue #87 for a bit more detail.
filegroup(
    name = "provider",
    provides = {"wibble": ":provided"},
)

genrule(
    name = "provided",
    outs = ["provided.txt"],
    cmd = 'echo "hello" > "$OUT"',
)

gentest(
    name = "require_test",
    srcs = [":provider"],
    outs = ["require_test.txt"],
    cmd = 'cat $SRCS > "$OUT"',
    no_test_output = True,
    requires = ["wibble"],
    test_cmd = "[ \"`cat $TEST`\" == \"hello\" ]",
)
