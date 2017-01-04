filegroup(
    name = 'please',
    srcs = ['//src:please'],
)

filegroup(
    name = 'all_tools',
    srcs = [
        '//src/cache/main:cache_cleaner',
        '//src/cache/server:http_cache_server_bin',
        '//src/cache/server:rpc_cache_server_bin',
        '//tools/jarcat',
        '//tools/javac_worker',
        '//tools/junit_runner',
        '//tools/linter',
        '//tools/please_diff_graphs',
        '//tools/please_go_test',
        '//tools/please_maven',
        '//tools/please_pex',
    ],
    deps = [
        '//:please',
    ],
)

filegroup(
    name = 'version',
    srcs = ['VERSION'],
    visibility = ['PUBLIC'],
)
