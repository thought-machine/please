go_library(
    name = 'metrics',
    srcs = ['prometheus.go'],
    deps = [
        '//src/core',
        '//third_party/go:logging',
        '//third_party/go:prometheus',
        '//third_party/go:shlex',
    ],
    visibility = ['PUBLIC'],
)

go_test(
    name = 'prometheus_test',
    srcs = ['prometheus_test.go'],
    flaky = True,
    deps = [
        ':metrics',
        '//third_party/go:testify',
    ],
)
