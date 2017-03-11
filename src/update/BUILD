go_library(
    name = 'update',
    srcs = glob(['*.go'], excludes = ['*_test.go']),
    deps = [
        '//src/cli',
        '//src/core',
        '//third_party/go:logging',
        '//third_party/go:semver',
    ],
    visibility = ['PUBLIC'],
)

go_test(
    name = 'update_test',
    srcs = ['update_test.go'],
    data = [
        ':test_tarball',
        '//src:please',
    ],
    deps = [
        ':update',
        '//src/cli',
        '//third_party/go:logging',
        '//third_party/go:testify',
    ],
)

tarball(
    name = 'test_tarball',
    srcs = [
        '//src:please',
    ],
    out = 'please_test.tar.gz',
    subdir = 'please',
)
