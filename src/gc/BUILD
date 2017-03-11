subinclude('//build_defs:go_bindata')

go_bindata(
    name = 'rewrite',
    srcs = ['rewrite.py'],
)

go_library(
    name = 'gc',
    srcs = [
        'gc.go',
        ':rewrite',
    ],
    deps = [
        '//src/core',
        '//src/parse',
        '//third_party/go:logging',
        '//third_party/go:prompter',
    ],
    visibility = ['PUBLIC'],
)

go_test(
    name = 'gc_test',
    srcs = ['gc_test.go'],
    deps = [
        ':gc',
        '//third_party/go:testify',
    ],
)

go_test(
    name = 'rewrite_test',
    srcs = ['rewrite_test.go'],
    data = [
        'test_data',
        '//src/parse/cffi:all_engines',
    ],
    labels = [
        'python2',
        'python3',
        'pypy',
    ],
    deps = [
        ':gc',
        '//third_party/go:testify',
    ],
)
