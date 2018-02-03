go_library(
    name = 'asp',
    srcs = glob(['*.go'], excludes = [
        '*_test.go',
        '*.bindata.go',
    ]),
    deps = [
        '//src/cli',
        '//src/core',
        '//third_party/go:logging',
        '//third_party/go:participle',
    ],
    visibility = ['//src/parse/...'],
)

go_test(
    name = 'parser_test',
    srcs = ['parser_test.go'],
    data = ['test_data'],
    deps = [
        ':asp',
        '//third_party/go:testify',
    ],
)

go_test(
    name = 'lexer_test',
    srcs = ['lexer_test.go'],
    deps = [
        ':asp',
        '//third_party/go:participle',
        '//third_party/go:testify',
    ],
)

go_test(
    name = 'interpreter_test',
    srcs = ['interpreter_test.go'],
    data = ['test_data'],
    deps = [
        ':asp',
        '//src/core',
        '//src/parse/asp/builtins',
        '//third_party/go:testify',
    ],
)

go_test(
    name = 'logging_test',
    srcs = ['logging_test.go'],
    data = ['test_data'],
    deps = [
        ':asp',
        '//src/core',
        '//src/parse/asp/builtins',
        '//third_party/go:logging',
        '//third_party/go:testify',
    ],
)
