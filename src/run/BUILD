go_library(
    name = 'run',
    srcs = glob(['*.go'], excludes=['*_test.go']),
    deps = [
        '//src/core',
        '//src/build',
	'//third_party/go:logging',
    ],
    visibility = ['PUBLIC'],
)
