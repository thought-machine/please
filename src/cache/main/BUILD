go_binary(
    name = 'cache_cleaner',
    main = 'cache_cleaner.go',
    deps = [
        '//src/cache/tools',
        '//src/output',
        '//third_party/go:logging',
        '//third_party/go:humanize',
    ],
    visibility = ['PUBLIC'],
)

go_test(
    name = 'cache_cleaner_test',
    srcs = ['cache_cleaner_test.go', 'cache_cleaner.go'],
    deps = [
        '//src/cache/tools',
        '//src/output',
        '//third_party/go:logging',
        '//third_party/go:humanize',
        '//third_party/go:testify',
    ],
)
