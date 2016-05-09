subinclude('//build_defs:go_bindata')

go_library(
    name = 'utils',
    srcs = glob(['*.go'], excludes=['wrapper_script.go', '*_test.go']) + [':wrapper_script'],
    deps = [
        '//src/core',
        '//third_party/go:prompter',
        '//third_party/go:logging',
    ],
    visibility = ['PUBLIC'],
)

go_bindata(
    name = 'wrapper_script',
    srcs = ['//src/misc:wrapper_script'],
    prefix = 'src/misc',
)

go_test(
    name = 'utils_test',
    srcs = ['utils_test.go'],
    data = ['test_data/stdin.txt'],
    deps = [
        ':utils',
        '//third_party/go:testify',
    ],
)
