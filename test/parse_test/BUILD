# Test that we can reasonably gracefully handle duplicate targets.

genrule(
    name = 'duplicate_target',
    cmd = 'echo hello',
)

duplicate = False

try:
    genrule(
        name = 'duplicate_target',
        cmd = 'echo goodbye',
    )
except DuplicateTargetError:
    duplicate = True

if not duplicate:
    raise ParseError('Should have found a duplicate target')

# Simulates a code generating rule to test the require / provide mechanism.
genrule(
    name = 'test_require_py',
    outs = ['test_require.py'],
    cmd = 'touch $OUT',
    test_only = True,
)

genrule(
    name = 'test_require_go',
    outs = ['test_require.go'],
    cmd = 'touch $OUT',
    test_only = True,
)

filegroup(
    name = 'test_require',
    srcs = [
        ':test_require_go',
        ':test_require_py',
    ],
    provides = {
        'py': ':test_require_py',
        'go': ':test_require_go',
    },
    test_only = True,
    deps = [
        ':test_require_go',
        ':test_require_py',
    ],
)

python_test(
    name = 'require_provide_test',
    srcs = ['require_provide_test.py'],
    deps = [
        ':test_require',
    ],
)

# Test for adding additional outputs to a target.
genrule(
    name = '_gen_output_name',
    cmd = 'echo test_additional_output.txt',
    post_build = lambda _, output: add_out(
        '_gen_output',
        ''.join(output).strip(),
    ),
)

genrule(
    name = '_gen_output',
    cmd = 'echo -n "kittens" > $OUT',
    deps = [
        ':_gen_output_name',
    ],
)

cgo_test(
    name = 'additional_output_test',
    srcs = ['additional_output_test.go'],
    data = [':_gen_output'],
)
