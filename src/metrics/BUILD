go_library(
    name = 'metrics',
    srcs = ['prometheus.go'],
    visibility = ['PUBLIC'],
    deps = [
        '//src/core',
        '//third_party/go:logging',
        '//third_party/go:prometheus',
    ],
)

go_test(
    name = 'prometheus_test',
    srcs = ['prometheus_test.go'],
    deps = [
        ':metrics',
        '//third_party/go:testify',
    ],
)
