genrule(
    name = 'version',
    srcs = {
        'go': ['version.go'],
        'version': ['//:version'],
    },
    outs = ['versioned.go'],
    cmd = 'sed "s/1.0.9999/`cat $SRCS_VERSION`/" $SRCS_GO > $OUT',
)

go_library(
    name = 'core',
    srcs = glob(['*.go'], excludes = [
        '*_test.go',
        'version.go',
    ]) + [':version'],
    deps = [
        '//src/cli',
        '//third_party/go:gcfg',
        '//third_party/go:logging',
        '//third_party/go:queue',
        '//third_party/go:semver',
    ],
    visibility = ['PUBLIC'],
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
    data = ['test_data'],
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
        '//third_party/go:testify',
    ],
)

go_test(
    name = 'utils_test',
    srcs = ['utils_test.go'],
    deps = [
        ':core',
        '//src/cli',
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

go_test(
    name = 'glob_test',
    srcs = ['glob_test.go'],
    data = glob(['test_data/**/*.txt']),
    deps = [
        ':core',
        '//third_party/go:testify',
    ],
)

go_test(
    name = 'build_label_test',
    srcs = ['build_label_test.go'],
    deps = [
        ':core',
        '//third_party/go:testify',
    ],
)

go_test(
    name = 'test_results_test',
    srcs = ['test_results_test.go'],
    deps = [
        ':core',
        '//third_party/go:testify',
    ],
)
