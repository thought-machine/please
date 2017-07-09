go_library(
    name = 'zip_writer',
    srcs = ['zip_writer.go'],
    out = 'jarcat.a',
    deps = [
        '//third_party/go:logging',
        '//third_party/go/zip',
    ],
    visibility = ['//tools/please_pex:all'],
)

go_binary(
    name = 'jarcat',
    srcs = ['jarcat.go'],
    deps = [
        ':zip_writer',
        '//src/cli',
        '//third_party/go:logging',
        '//third_party/go/zip',
        '//tools/jarcat/tar',
    ],
    visibility = ['PUBLIC'],
)

go_test(
    name = 'zip_writer_test',
    srcs = ['zip_writer_test.go'],
    data = glob(['test_data/*.zip']),
    deps = [
        ':zip_writer',
        '//third_party/go/zip',
    ],
)
