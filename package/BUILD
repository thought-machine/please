subinclude("//build_defs:version")

subinclude("//build_defs:signing")

# This is the master list of files that get installed.
filegroup(
    name = "installed_files",
    srcs = [
        "//src:please",
        "//tools/go_buildid_replacer",
        "//tools/jarcat",
        "//tools/java:javac_worker",
        "//tools/java:junit_runner",
        "//tools/please_go_filter",
        "//tools/please_go_test",
        "//tools/please_maven",
        "//tools/please_pex",
    ],
)

signed_tarball(
    name = "please_tarball",
    srcs = [":installed_files"],
    out = "please_%s.tar.gz" % VERSION,
    subdir = "please",
)

signed_tarball(
    name = "please_tarball_xz",
    srcs = [":installed_files"],
    out = "please_%s.tar.xz" % VERSION,
    subdir = "please",
    xzip = True,
)

signed_tarball(
    name = "servers_tarball",
    srcs = [
        "//tools/cache:http_cache_server",
        "//tools/cache:rpc_cache_server",
    ],
    out = "please_servers_%s.tar.gz" % VERSION,
    subdir = "please",
)

filegroup(
    name = "tarballs",
    srcs = [
        ":please_tarball",
        ":please_tarball_xz",
        ":servers_tarball",
    ],
    labels = ["hlink:plz-out/pkg/${OS}_${ARCH}"],
)

filegroup(
    name = "signed_tarballs",
    srcs = [
        ":please_tarball_signed",
        ":please_tarball_xz_signed",
        ":servers_tarball_signed",
    ],
    labels = ["manual"],
)

# This generates a stamped version of the various inputs which will have a unique name.
# We don't really use it here, it's for testing purposes really.
hash_filegroup(
    name = "hash_files",
    srcs = [":installed_files"],
)

filegroup(
    name = "signer",
    srcs = ["//tools/release_signer"],
    labels = ["hlink:plz-out/pkg"],
)
