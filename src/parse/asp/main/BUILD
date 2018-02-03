go_binary(
    name = 'asp',
    srcs = ['main.go'],
    deps = [
        '//src/cli',
        '//src/core',
        '//src/parse/asp',
        '//src/parse/asp/builtins',
        '//third_party/go:logging',
        '//third_party/go:participle',
        '//third_party/go:spew',
    ],
)

go_binary(
    name = 'compiler',
    srcs = ['compiler.go'],
    deps = [
        '//src/cli',
        '//src/parse/asp',
        '//third_party/go:logging',
    ],
    visibility = ['//src/parse/asp/builtins:all'],
)
