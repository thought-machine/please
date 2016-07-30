go_library(
    name = 'run',
    srcs = glob(['*.go'], excludes = ['*_test.go']),
    visibility = ['PUBLIC'],
    deps = [
        '//src/build',
        '//src/core',
        '//src/output',
        '//third_party/go:logging',
    ],
)
