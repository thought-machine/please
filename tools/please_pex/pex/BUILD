go_bindata(
    name = 'main_files',
    srcs = ['//tools/please_pex:pex_data'],
    input_dir = 'tools/please_pex',
    prefix = 'tools/please_pex',
)

go_library(
    name = 'pex',
    srcs = [
        'pex.go',
        ':main_files',
    ],
    deps = [
        '//tools/jarcat/zip',
    ],
    visibility = ['//tools/please_pex:all'],
)
