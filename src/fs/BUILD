go_library(
    name = 'fs',
    srcs = [
        'fs.go',
        'glob.go',
        'hash.go',
    ],
    deps = ['//third_party/go:godirwalk'],
    visibility = ['PUBLIC'],
)

go_test(
    name = 'fs_test',
    srcs = ['fs_test.go'],
    data = ['test_data'],
    deps = [
        ':fs',
        '//third_party/go:testify',
    ],
)

go_test(
    name = 'hash_test',
    srcs = ['hash_test.go'],
    data = ['test_data'],
    deps = [
        ':fs',
        '//third_party/go:testify',
    ],
)
