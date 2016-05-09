go_library(
    name = 'query',
    srcs = glob(['*.go'], excludes=['*_test.go']),
    deps = [
        '//src/build',
        '//src/core',
        '//src/utils',
	'//third_party/go:logging',
    ],
    visibility = ['PUBLIC'],
)

go_test(
    name = 'print_test',
    srcs = ['print_test.go'],
    deps = [
        ':query',
        '//src/core',
    ],
)

go_test(
    name = 'graph_test',
    srcs = ['graph_test.go'],
    deps = [
        ':query',
        '//src/core',
        '//third_party/go:testify',
    ],
)
