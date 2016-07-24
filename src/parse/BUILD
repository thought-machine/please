subinclude('//build_defs:go_bindata')

cgo_library(
    name = 'parse',
    srcs = glob(
        [
            '*.go',
            'interpreter.*',
        ],
        excludes = [
            '*_test.go',
            'builtin_rules.go',
        ],
    ) + [':builtin_rules'],
    visibility = ['PUBLIC'],
    deps = [
        ':builtin_rules',
        '//src/core',
        '//third_party/go:gcfg',
        '//third_party/go:levenshtein',
        '//third_party/go:logging',
        '//third_party/go:osext',
    ],
)

genrule(
    name = 'type_checked_rules',
    srcs = ['type_checking.py'] + glob(['rules/*.py']),
    outs = [basename(x) for x in glob(['rules/*.py'])],
    cmd = 'python $SRCS',
)

go_bindata(
    name = 'builtin_rules',
    srcs = [':type_checked_rules'],
    prefix = '${PKG}',
)

cgo_test(
    name = 'glob_test',
    srcs = ['glob_test.go'],
    data = glob(['test_data/**/*.txt']),
    deps = [
        ':parse',
        '//src/core',
        '//third_party/go:testify',
    ],
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
    data = glob([
        'test_data/**/TEST_BUILD',
        'test_data/**/test.py',
    ]),
    deps = [
        ':parse',
        '//third_party/go:testify',
    ],
)

cgo_test(
    name = 'suggest_test',
    srcs = ['suggest_test.go'],
    deps = [
        ':parse',
        '//third_party/go:testify',
    ],
)
