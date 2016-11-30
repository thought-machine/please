java_binary(
    name = 'junit_runner',
    main_class = 'build.please.test.TestMain',
    deps = [
        '//src/build/java/build/please/test:junit_runner',
    ],
    visibility = ['PUBLIC'],
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
    deps = [
        ':java',
        '//src/cli',
        '//third_party/go:logging',
        '//third_party/go/zip',
    ],
    visibility = ['PUBLIC'],
)

go_binary(
    name = 'please_maven',
    deps = [
        '//src/cli',
        '//third_party/go:logging',
    ],
    visibility = ['PUBLIC'],
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
