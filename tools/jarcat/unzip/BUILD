go_library(
    name = 'unzip',
    srcs = ['unzip.go'],
    deps = ['//third_party/go/zip'],
    visibility = ['//tools/jarcat:all'],
)

go_test(
    name = 'unzip_test',
    srcs = ['unzip_test.go'],
    data = ['test_data'],
    deps = [
        ':unzip',
        '//third_party/go:testify',
    ],
)
