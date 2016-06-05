go_library(
    name = 'tools',
    # We need to use things from syscall that vary by platform.
    srcs = ['tools_linux.go' if CONFIG.OS == 'linux' else 'tools.go'],
    visibility = ['//src/cache/...'],
)

go_binary(
    name = 'cache_cleaner',
    main = 'cache_cleaner.go',
    deps = [
        ':tools',
        '//src/output',
        '//third_party/go:logging',
        '//third_party/go:humanize',
    ],
    visibility = ['PUBLIC'],
)

cgo_test(
    name = 'cache_cleaner_test',
    srcs = ['cache_cleaner_test.go', ':cache_cleaner_platform'],
    deps = [
        '//src/output',
        '//third_party/go:logging',
        '//third_party/go:humanize',
        '//third_party/go:testify',
    ],
)
