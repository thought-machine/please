genrule(
    name = 'version',
    srcs = {
        'go': ['version.go'],
        'version': ['//:version'],
    },
    outs = ['versioned.go'],
    cmd = 'sed "s/<git>/`cat $SRCS_VERSION`/" $SRCS_GO > $OUT',
)

go_library(
    name = 'core',
    srcs = glob(
        ['*.go'],
        excludes = [
            '*_test.go',
            'version.go',
        ],
    ) + [':version'],
    visibility = ['PUBLIC'],
    deps = [
        '//third_party/go:gcfg',
        '//third_party/go:logging',
        '//third_party/go:queue',
    ],
)

go_test(
    name = 'config_test',
    srcs = ['config_test.go'],
    data = glob(['test_data/*.plzconfig']),
    deps = [
        ':core',
        '//third_party/go:testify',
    ],
)

go_test(
    name = 'label_parse_test',
    srcs = ['label_parse_test.go'],
    deps = [
        ':core',
    ],
)

go_test(
    name = 'graph_test',
    srcs = ['graph_test.go'],
    deps = [
        ':core',
        '//third_party/go:testify',
    ],
)

go_test(
    name = 'state_test',
    srcs = ['state_test.go'],
    deps = [
        ':core',
        '//third_party/go:testify',
    ],
)

go_test(
    name = 'build_target_test',
    srcs = ['build_target_test.go'],
    deps = [
        ':core',
        '//third_party/go:testify',
    ],
)

go_test(
    name = 'package_test',
    srcs = ['package_test.go'],
    deps = [
        ':core',
        '//third_party/go:testify',
    ],
)

go_test(
    name = 'build_env_test',
    srcs = ['build_env_test.go'],
    deps = [
        ':core',
    ],
)

go_test(
    name = 'utils_test',
    srcs = ['utils_test.go'],
    deps = [
        ':core',
        '//third_party/go:testify',
    ],
)

go_test(
    name = 'lock_test',
    srcs = ['lock_test.go'],
    deps = [
        ':core',
        '//third_party/go:testify',
    ],
)
