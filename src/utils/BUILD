subinclude('//build_defs:go_bindata')

go_library(
    name = 'utils',
    srcs = glob(['*.go'], excludes=['wrapper_script.go']) + [':wrapper_script'],
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
