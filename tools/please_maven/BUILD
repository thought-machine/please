go_library(
    name = 'maven',
    srcs = [
        'fetch.go',
        'pom.go',
        'print.go',
        'resolver.go',
    ],
    deps = [
        '//third_party/go:go-flags',
        '//third_party/go:logging',
        '//third_party/go:queue',
    ],
)

go_binary(
    name = 'please_maven',
    srcs = ['main.go'],
    deps = [
        ':maven',
        '//src/cli',
    ],
    visibility = ['PUBLIC'],
)

go_test(
    name = 'maven_test',
    srcs = ['maven_test.go'],
    data = ['test_data'],
    deps = [
        ':maven',
        '//src/cli',
        '//third_party/go:testify',
    ],
)

go_test(
    name = 'version_test',
    srcs = ['version_test.go'],
    deps = [
        ':maven',
        '//third_party/go:testify',
    ],
)
