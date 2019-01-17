subinclude("//build_defs:version")

# This is the master list of files that get installed.
filegroup(
    name = "installed_files",
    srcs = [
        "//src:please",
        "//tools/build_langserver",
        "//tools/go_buildid_replacer",
        "//tools/jarcat",
        "//tools/java:javac_worker",
        "//tools/java:junit_runner",
        "//tools/please_go_filter",
        "//tools/please_go_test",
        "//tools/please_maven",
        "//tools/please_pex",
        "//tools/sandbox:please_sandbox",
    ],
)

tarball(
    name = "please_tarball",
    srcs = [":installed_files"],
    out = "please_%s.tar.gz" % VERSION,
    subdir = "please",
)

tarball(
    name = "please_tarball_xz",
    srcs = [":installed_files"],
    out = "please_%s.tar.xz" % VERSION,
    subdir = "please",
    xzip = True,
)

tarball(
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
    name = "signer",
    srcs = ["//tools/release_signer"],
    labels = ["hlink:plz-out/pkg"],
)
