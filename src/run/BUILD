go_library(
    name = 'run',
    srcs = ['run_step.go'],
    deps = [
        '//src/build',
        '//src/core',
        '//src/output',
        '//third_party/go:errgroup',
        '//third_party/go:logging',
    ],
    visibility = ['PUBLIC'],
)

go_test(
    name = 'run_test',
    srcs = ['run_test.go'],
    data = ['test_data'],
    deps = [
        ':run',
        '//third_party/go:testify',
    ],
)
