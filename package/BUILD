fpm_deb(
    name = 'please',
    version = CONFIG.PLZ_VERSION,
    files = {
        '/opt/please/please': '//src:please',
        '/opt/please/please_pex': '//tools/please_pex',
        '/opt/please/please_maven': '//tools/please_maven',
        '/opt/please/javac_worker': '//tools/javac_worker',
        '/opt/please/junit_runner.jar': '//tools/junit_runner',
        '/opt/please/cache_cleaner': '//tools/cache_cleaner',
        '/opt/please/jarcat': '//tools/jarcat',
        '/opt/please/please_diff_graphs': '//tools/please_diff_graphs',
        '/opt/please/please_go_test': '//tools/please_go_test',
        '/opt/please/please_build_linter': '//tools/linter',
        '/opt/please/libplease_parser_pypy.so': '//src/parse/cffi:please_parser_pypy',
        '/opt/please/libplease_parser_python2.so': '//src/parse/cffi:please_parser_python2',
        '/opt/please/libplease_parser_python3.so': '//src/parse/cffi:please_parser_python3',
    },
    links = {
        '/usr/bin/plz': '/opt/please/please',
        '/usr/bin/plz_diff_graphs': '/opt/please/please_diff_graphs',
    },
)

fpm_deb(
    name = 'plz_rpc_cache_server',
    package_name = 'plz-rpc-cache-server',
    version = CONFIG.PLZ_VERSION,
    files = {
        '/usr/bin/plz_rpc_cache_server': '//src/cache/server:rpc_cache_server_bin',
    },
)

fpm_deb(
    name = 'plz_http_cache_server',
    package_name = 'plz-http-cache-server',
    version = CONFIG.PLZ_VERSION,
    files = {
        '/usr/bin/plz_http_cache_server': '//src/cache/server:http_cache_server_bin',
    },
)

fpm_deb(
    name = 'plz_build_linter',
    package_name = 'please-build-lint',
    version = CONFIG.PLZ_VERSION,
    files = {
        '/opt/please/please_build_linter': '//tools/linter',
    },
)

tarball(
    name = 'please_tarball',
    srcs = [
        '//src:please',
        '//src/parse/cffi:all_engines',
        '//tools/cache_cleaner',
        '//tools/jarcat',
        '//tools/javac_worker',
        '//tools/junit_runner',
        '//tools/linter',
        '//tools/please_diff_graphs',
        '//tools/please_go_test',
        '//tools/please_maven',
        '//tools/please_pex',
    ],
    out = 'please_%s.tar.gz' % CONFIG.PLZ_VERSION,
    subdir = 'please',
)

tarball(
    name = 'servers_tarball',
    srcs = [
        '//src/cache/server:http_cache_server_bin',
        '//src/cache/server:rpc_cache_server_bin',
    ],
    out = 'please_servers_%s.tar.gz' % CONFIG.PLZ_VERSION,
    subdir = 'please',
)

filegroup(
    name = 'tarballs',
    srcs = [
        ':please_tarball',
        ':servers_tarball',
    ],
)

# This generates a stamped version of the various inputs which will have a unique name.
# We don't really use it here, it's for testing purposes really.
hash_filegroup(
    name = 'hash_tools',
    srcs = [
        '//src:please',
        '//src/parse/cffi:all_engines',
        '//tools/cache_cleaner',
        '//tools/jarcat',
        '//tools/javac_worker',
        '//tools/junit_runner',
        '//tools/linter',
        '//tools/please_diff_graphs',
        '//tools/please_go_test',
        '//tools/please_maven',
        '//tools/please_pex',
    ],
)
