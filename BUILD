filegroup(
    name = 'please',
    srcs = ['//src:please'],
)

filegroup(
    name = 'all_tools',
    srcs = [
        '//src/build/go:please_go_test',
        '//src/build/java:jarcat',
        '//src/build/java:junit_runner',
        '//src/build/java:please_maven',
        '//src/build/python:please_pex',
        '//src/cache/main:cache_cleaner',
        '//src/cache/server:http_cache_server_bin',
        '//src/cache/server:rpc_cache_server_bin',
        '//src/misc:plz_diff_graphs',
        '//src/parse/cffi:all_engines',
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
