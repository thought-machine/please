subinclude('//build_defs:go_bindata')

cgo_library(
    name = 'parse',
    srcs = glob(['parse_step.go', 'suggest.go', 'interpreter.*']) + [':builtin_rules'],
    env = {'CGO_LDFLAGS': '-L${TMP_DIR}/${PKG}/rules'},
    deps = [
        ':defs_hdr',
        ':builtin_rules',
        '//src/core',
        '//third_party/go:logging',
        '//third_party/go:gcfg',
        '//third_party/go:levenshtein',
    ],
    visibility = ['PUBLIC'],
)

filegroup(
    name = 'defs_hdr',
    srcs = ['defs.h'],
    visibility = ['//src/parse/cffi:all'],
)

go_bindata(
    name = 'builtin_rules',
    srcs = glob(['rules/*.py'], excludes=['embedded_parser.py']) + [':embedded_parser'],
    prefix = '${PKG}/rules',
)

genrule(
    name = 'embedded_parser',
    srcs = ['//src/parse/cffi:embedded_parser'],
    outs = ['rules/embedded_parser.py'],
    cmd = 'cp $SRC $OUT',
)

cgo_test(
    name = 'glob_test',
    srcs = ['glob_test.go'],
    deps = [
        ':parse',
        '//src/core',
        '//third_party/go:testify',
    ],
    data = glob(['test_data/**/*.txt']),
)

cgo_test(
    name = 'parse_step_test',
    srcs = ['parse_step_test.go'],
    deps = [
        ':parse',
        '//src/core',
        '//third_party/go:testify',
    ],
)

cgo_test(
    name = 'interpreter_test',
    srcs = ['interpreter_test.go'],
    deps = [
        ':parse',
        '//third_party/go:testify',
    ],
    data = glob(['test_data/**/TEST_BUILD', 'test_data/**/test.py']),
)

cgo_test(
    name = 'suggest_test',
    srcs = ['suggest_test.go'],
    deps = [
        ':parse',
        '//third_party/go:testify',
    ],
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
cgo_test(
    name = 'additional_output_test',
    srcs = ['additional_output_test.go'],
    data = [':_gen_output'],
)
