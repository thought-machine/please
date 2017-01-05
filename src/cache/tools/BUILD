go_library(
    name = 'tools',
    # We need to use things from syscall that vary by platform.
    srcs = ['tools_linux.go' if (CONFIG.OS == 'linux') else 'tools.go'] + ['hash.go'],
    visibility = [
        '//src/cache/...',
        '//tools/cache_cleaner:all',
    ],
)

go_test(
    name = 'hash_test',
    srcs = ['hash_test.go'],
    deps = [
        ':tools',
        '//third_party/go:testify',
    ],
)
