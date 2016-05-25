subinclude('//build_defs:go_bindata')

cgo_library(
    name = 'parse',
    srcs = glob(['*.go', 'interpreter.*'], excludes=['*_test.go', 'builtin_rules.go']) + [':builtin_rules'],
    deps = [
        ':builtin_rules',
        '//src/core',
        '//third_party/go:logging',
        '//third_party/go:gcfg',
        '//third_party/go:levenshtein',
        '//third_party/go:osext',
    ],
    visibility = ['PUBLIC'],
)

go_bindata(
    name = 'builtin_rules',
    srcs = glob(['rules/*.py']),
    prefix = '${PKG}/rules',
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
