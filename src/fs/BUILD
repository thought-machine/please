go_library(
    name = 'fs',
    srcs = [
        'fs.go',
        'glob.go',
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
