subinclude('//build_defs:go_bindata')

cgo_library(
    name = 'parse',
    srcs = glob(['parse_step.go', 'suggest.go', 'glob.go', 'interpreter.*']) + [':builtin_rules'],
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
