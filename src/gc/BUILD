go_library(
    name = 'gc',
    srcs = glob(['*.go'], excludes = ['*_test.go']),
    deps = [
        '//src/core',
        '//third_party/go:logging',
    ],
    visibility = ['PUBLIC'],
)

go_test(
    name = 'gc_test',
    srcs = ['gc_test.go'],
    deps = [
        ':gc',
        '//third_party/go:testify',
    ],
)
