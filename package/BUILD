fpm_deb(
    name = 'please',
    version = CONFIG.PLZ_VERSION,
    files = {
        '/opt/please/please': '//src:please',
        '/opt/please/please_pex': '//src/build/python:please_pex',
        '/opt/please/please_maven': '//src/build/java:please_maven',
        '/opt/please/junit_runner.jar': '//src/build/java:junit_runner',
        '/opt/please/cache_cleaner': '//src/cache/main:cache_cleaner',
        '/opt/please/jarcat': '//src/build/java:jarcat',
        '/opt/please/please_diff_graphs': '//src/misc:plz_diff_graphs',
        '/opt/please/please_go_test': '//src/build/go:please_go_test',
        '/opt/please/libplease_parser_pypy.so': '//src/parse/cffi:please_parser_pypy',
        '/opt/please/libplease_parser_python2.so': '//src/parse/cffi:please_parser_python2',
        '/opt/please/libplease_parser_python3.so': '//src/parse/cffi:please_parser_python3',
    },
    links = {
        '/usr/bin/plz': '/opt/please/please',
        '/usr/bin/plz_diff_graphs': '/opt/please/please_diff_graphs',
    },
    labels = ['manual'],
)

fpm_deb(
    name = 'plz_rpc_cache_server',
    package_name = 'plz-rpc-cache-server',
    version = CONFIG.PLZ_VERSION,
    files = {
        '/usr/bin/plz_rpc_cache_server': '//src/cache/server:rpc_cache_server_bin',
    },
    labels = ['manual'],
)

fpm_deb(
    name = 'plz_http_cache_server',
    package_name = 'plz-http-cache-server',
    version = CONFIG.PLZ_VERSION,
    files = {
        '/usr/bin/plz_http_cache_server': '//src/cache/server:http_cache_server_bin',
    },
    labels = ['manual'],
)

# Remove the .pex suffix from this guy
genrule(
    name = 'please_pex',
    srcs = ['//src/build/python:please_pex'],
    outs = ['please_pex'],
    cmd = 'cp $SRC $OUT',
    binary = True,
)

tarball(
    name = 'please_tarball',
    srcs = [
        '//src:please',
        ':please_pex',
        '//src/build/java:junit_runner',
        '//src/cache/main:cache_cleaner',
        '//src/build/java:jarcat',
        '//src/build/java:please_maven',
        '//src/misc:plz_diff_graphs',
        '//src/build/go:please_go_test',
        '//src/parse/cffi:all_engines',
    ],
    subdir = 'please',
    out = 'please_%s.tar.gz' % CONFIG.PLZ_VERSION,
)

tarball(
    name = 'servers_tarball',
    srcs = [
        '//src/cache/server:http_cache_server_bin',
        '//src/cache/server:rpc_cache_server_bin',
    ],
    subdir = 'please',
    out = 'please_servers_%s.tar.gz' % CONFIG.PLZ_VERSION,
)

filegroup(
    name = 'tarballs',
    deps = [':please_tarball', ':servers_tarball'],
)
