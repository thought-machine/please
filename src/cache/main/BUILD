go_binary(
    name = 'cache_cleaner',
    main = 'cache_cleaner.go',
    deps = [
        '//src/cache/tools',
        '//src/cli',
        '//third_party/go:humanize',
        '//third_party/go:logging',
    ],
    visibility = ['PUBLIC'],
)

go_test(
    name = 'cache_cleaner_test',
    srcs = [
        'cache_cleaner.go',
        'cache_cleaner_test.go',
    ],
    deps = [
        '//src/cache/tools',
        '//src/cli',
        '//third_party/go:humanize',
        '//third_party/go:logging',
        '//third_party/go:testify',
    ],
)
