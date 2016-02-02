filegroup(
    name = 'please',
    srcs = ['//src:please'],
)

filegroup(
    name = 'all_tools',
    srcs = [
        '//src/build/python:please_pex',
        '//src/build/java:junit_runner',
        '//src/cache/tools:cache_cleaner',
        '//src/cache/server:http_cache_server_bin',
        '//src/cache/server:rpc_cache_server_bin',
        '//src/build/java:jarcat',
        '//src/build/java:please_maven',
        '//src/misc:plz_diff_graphs',
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
