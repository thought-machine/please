cgo_library(
    name = 'parse',
    srcs = glob(['*.go', '*.h', '*.c', 'python_interface.*'], excludes=['builtin_rules.go', 'go_rules_test_*', '*_test.go']) + [':builtin_rules'],
    env = {'CGO_LDFLAGS': '-L${TMP_DIR}/${PKG}/rules'},
    deps = [
        ':builtin_rules',
        '//src/core',
        '//third_party/go:logging',
        '//third_party/go:gcfg',
    ],
    visibility = ['PUBLIC'],
)


genrule(
    name = 'builtin_rules',
    srcs = glob(['rules/*.py'], excludes=['embedded_parser.py']) + ['//src/parse/cffi:embedded_parser'],
    outs = ['builtin_rules.go'],
    # Massive hack since symlinks don't seem to be working with go-bindata here.
    # Not sure why, they have a resolved issue for them...
    cmd = 'for i in `find -L $PKG -type f`; do cp $i tmp && rm $i && mv tmp $i; done && '
          '$(exe //third_party/go:go-bindata) -o $OUT -nomemcopy -pkg parse -prefix ${PKG}/rules ${PKG}/rules',
    deps = [
        '//third_party/go:go-bindata',
    ]
)

go_test(
    name = 'glob_test',
    srcs = ['glob_test.go'],
    deps = [
        ':parse',
        '//src/core',
    ],
    data = glob(['test_data/**/*.txt']),
)

go_test(
    name = 'parse_step_test',
    srcs = ['parse_step_test.go'],
    deps = [
        ':parse',
        '//src/core',
        '//third_party/go:testify',
    ],
)

go_test(
    name = 'interpreter_test',
    srcs = ['interpreter_test.go'],
    deps = [
        ':parse',
        '//third_party/go:testify',
    ],
    data = glob(['test_data/**/TEST_BUILD', 'test_data/**/test.py']),
)

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
    srcs = [':test_require_py', ':test_require_go'],
    deps = [':test_require_py', ':test_require_go'],
    provides = {
        'py': ':test_require_py',
        'go': ':test_require_go',
    },
    test_only = True,
)
python_test(
    name = 'require_provide_test',
    srcs = ['require_provide_test.py'],
    deps = [
        ':test_require',
        '//src/build/python:bootstrap_pexer',
    ],
)

# Test for adding additional outputs to a target.
genrule(
    name = '_gen_output_name',
    cmd = 'echo test_additional_output.txt',
    post_build = lambda _, output: add_out('_gen_output', ''.join(output).strip()),
)
genrule(
    name = '_gen_output',
    cmd = 'echo -n "kittens" > $OUT',
    deps = [
        ':_gen_output_name',
    ],
)
go_test(
    name = 'additional_output_test',
    srcs = ['additional_output_test.go'],
    data = [':_gen_output'],
)
