go_library(
    name = 'build',
    srcs = glob(['*.go'], excludes=['*_test.go']),
    deps = [
        '//src/core',
        '//src/parse',
        '//src/cache',
	'//third_party/go:logging',
    ],
    visibility = ['PUBLIC'],
)

go_test(
    name = 'command_replacements_test',
    srcs = ['command_replacements_test.go'],
    deps = [
        ':build',
        '//third_party/go:testify',
    ]
)

go_test(
    name = 'incrementality_test',
    srcs = ['incrementality_test.go'],
    deps = [
        ':build',
    ],
)

go_test(
    name = 'build_step_test',
    srcs = ['build_step_test.go'],
    deps = [
        ':build',
        '//src/core',
        '//src/output',
        '//third_party/go:testify',
    ],
    mocks = {
        'parse': '//src/mock:parse',
    },
    data = ['test_data'],
)
