subinclude('//build_defs:engines')

go_library(
    name = 'parse',
    srcs = [
        'init.go',
        'parse_step.go',
        'rules.go',
        'suggest.go',
    ],
    deps = [
        '//src/core',
        '//src/parse/asp',
        '//src/parse/rules',
        '//src/utils',
        '//third_party/go:gcfg',
        '//third_party/go:logging',
    ],
    visibility = ['PUBLIC'],
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
    name = 'suggest_test',
    srcs = ['suggest_test.go'],
    deps = [
        ':parse',
        '//third_party/go:testify',
    ],
)
