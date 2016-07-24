java_binary(
    name = 'junit_runner',
    main_class = 'net.thoughtmachine.please.test.TestMain',
    visibility = ['PUBLIC'],
    deps = [
        '//src/build/java/net/thoughtmachine/please/test:junit_runner',
    ],
)

go_library(
    name = 'java',
    srcs = ['zip_writer.go'],
    deps = [
        '//third_party/go:logging',
        '//third_party/go/zip',
    ],
)

go_binary(
    name = 'jarcat',
    visibility = ['PUBLIC'],
    deps = [
        ':java',
        '//src/output',
        '//third_party/go:logging',
        '//third_party/go/zip',
    ],
)

go_binary(
    name = 'please_maven',
    visibility = ['PUBLIC'],
    deps = [
        '//src/output',
        '//third_party/go:logging',
    ],
)

go_test(
    name = 'zip_writer_test',
    srcs = ['zip_writer_test.go'],
    data = glob(['test_data/*.zip']),
    deps = [
        ':java',
        '//third_party/go/zip',
    ],
)
