go_binary(
    name = 'jarcat',
    srcs = ['main.go'],
    deps = [
        '//src/cli',
        '//third_party/go:logging',
        '//tools/jarcat/tar',
        '//tools/jarcat/unzip',
        '//tools/jarcat/zip',
    ],
    visibility = ['PUBLIC'],
)

go_binary(
    name = 'jarcat_unzip',
    srcs = ['unzip_main.go'],
    deps = [
        '//tools/jarcat/unzip',
    ],
    visibility = ['//third_party/go:all'],
)
