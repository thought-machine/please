fpm_deb(
    name = 'please',
    version = CONFIG.PLZ_VERSION,
    files = {
        '/opt/please/please': '//src:please',
        '/opt/please/please_pex': '//tools/please_pex',
        '/opt/please/please_maven': '//tools/please_maven',
        '/opt/please/javac_worker': '//src/build/java:javac_worker',
        '/opt/please/junit_runner.jar': '//src/build/java:junit_runner',
        '/opt/please/cache_cleaner': '//src/cache/main:cache_cleaner',
        '/opt/please/jarcat': '//tools/jarcat',
        '/opt/please/please_diff_graphs': '//src/misc:please_diff_graphs',
        '/opt/please/please_go_test': '//tools/please_go_test',
        '/opt/please/please_build_linter': '//src/lint:please_build_linter',
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
        '/opt/please/please_build_linter': '//src/lint:please_build_linter',
    },
)

tarball(
    name = 'please_tarball',
    srcs = [
        '//src:please',
        '//src/build/java:javac_worker',
        '//src/build/java:junit_runner',
        '//src/cache/main:cache_cleaner',
        '//src/lint:please_build_linter',
        '//src/misc:please_diff_graphs',
        '//src/parse/cffi:all_engines',
        '//tools/jarcat',
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
    deps = [
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
        '//src/build/java:junit_runner',
        '//src/cache/main:cache_cleaner',
        '//src/lint:please_build_linter',
        '//src/misc:please_diff_graphs',
        '//src/parse/cffi:all_engines',
        '//tools/jarcat',
        '//tools/please_go_test',
        '//tools/please_maven',
        '//tools/please_pex',
    ],
)
