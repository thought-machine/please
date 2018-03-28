subinclude('//build_defs:version')
subinclude('//build_defs:fpm')
subinclude('//build_defs:signing')

# This is the master list of files that get installed.
filegroup(
    name = 'installed_files',
    srcs = [
        '//src:please',
        '//tools/jarcat',
        '//tools/java:javac_worker',
        '//tools/java:junit_runner',
        '//tools/please_diff_graphs',
        '//tools/please_go_filter',
        '//tools/please_go_test',
        '//tools/please_maven',
        '//tools/please_pex',
    ],
)

signed_fpm_deb(
    name = 'please',
    version = VERSION,
    dirs = {'/opt/please': [':installed_files']},
    links = {
        '/usr/bin/plz': '/opt/please/please',
        '/usr/bin/plz_diff_graphs': '/opt/please/please_diff_graphs',
    },
)

signed_fpm_deb(
    name = 'plz_rpc_cache_server',
    package_name = 'plz-rpc-cache-server',
    version = VERSION,
    files = {
        '/usr/bin/plz_rpc_cache_server': '//tools/cache:rpc_cache_server',
    },
)

signed_fpm_deb(
    name = 'plz_http_cache_server',
    package_name = 'plz-http-cache-server',
    version = VERSION,
    files = {
        '/usr/bin/plz_http_cache_server': '//tools/cache:http_cache_server',
    },
)

signed_fpm_deb(
    name = 'please_sandbox',
    version = VERSION,
    files = {
        '/usr/bin/please_sandbox': '//tools/sandbox:please_sandbox',
    },
    # fpm builds us the deb, but we need to apply the suid bit separately.
    post_cmds = [
        'ar x $OUT',
        'rm $OUT',
        'tar -xzf data.tar.gz',
        'rm data.tar.gz',
        'tar -czf data.tar.gz --mode="u+s" --owner root --group root usr/bin/please_sandbox',
        'ar -rc $OUT debian-binary control.tar.gz data.tar.gz',
    ],
)

signed_tarball(
    name = 'please_tarball',
    srcs = [':installed_files'],
    out = 'please_%s.tar.gz' % VERSION,
    subdir = 'please',
)

signed_tarball(
    name = 'servers_tarball',
    srcs = [
        '//tools/cache:http_cache_server',
        '//tools/cache:rpc_cache_server',
    ],
    out = 'please_servers_%s.tar.gz' % VERSION,
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
    name = 'hash_files',
    srcs = [':installed_files'],
)
