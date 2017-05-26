go_library(
    name = 'run',
    srcs = glob(['*.go'], excludes = ['*_test.go']),
    deps = [
        '//src/build',
        '//src/core',
        '//src/output',
        '//third_party/go:errgroup',
        '//third_party/go:logging',
    ],
    visibility = ['PUBLIC'],
)
