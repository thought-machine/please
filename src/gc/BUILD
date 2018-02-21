go_library(
    name = 'gc',
    srcs = ['gc.go'],
    deps = [
        '//src/core',
        '//src/parse/asp',
        '//third_party/go:logging',
        '//third_party/go:prompter',
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

go_test(
    name = 'rewrite_test',
    srcs = ['rewrite_test.go'],
    data = [
        'test_data',
    ],
    deps = [
        ':gc',
        '//third_party/go:testify',
    ],
)
