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
    labels = ['manual'],
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
    binary = True,
    cmd = 'cp $SRC $OUT',
)

tarball(
    name = 'please_tarball',
    srcs = [
        ':please_pex',
        '//src:please',
        '//src/build/go:please_go_test',
        '//src/build/java:jarcat',
        '//src/build/java:junit_runner',
        '//src/build/java:please_maven',
        '//src/cache/main:cache_cleaner',
        '//src/misc:plz_diff_graphs',
        '//src/parse/cffi:all_engines',
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
        ':please_pex',
        '//src/build/java:junit_runner',
        '//src/cache/main:cache_cleaner',
        '//src/build/java:jarcat',
        '//src/build/java:please_maven',
        '//src/misc:plz_diff_graphs',
        '//src/build/go:please_go_test',
        '//src/parse/cffi:all_engines',
    ],
)
