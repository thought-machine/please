subinclude("//build_defs:version")

# This is the master list of files that get installed.
filegroup(
    name = "tools",
    srcs = [
        "//tools/build_langserver",
        "//tools/jarcat",
        "//tools/java:javac_worker",
        "//tools/java:junit_runner",
        "//tools/please_go_filter",
        "//tools/please_go_test",
        "//tools/please_pex",
        "//tools/sandbox:please_sandbox",
    ],
    binary = True,
    visibility = ["//src:tools"],
)

filegroup(
    name = "installed_files",
    srcs = [
        ":tools",
        "//src:please",
    ],
    binary = True,
    visibility = ["//:install"],
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

genrule(
    name = "please",
    srcs = ["//src:please"],
    outs = ["please_" + VERSION],
    binary = True,
    cmd = "mv $SRC $OUT",
    labels = ["hlink:plz-out/pkg/${OS}_${ARCH}"],
)

genrule(
    name = "jarcat",
    srcs = ["//tools/jarcat"],
    outs = ["jarcat_" + VERSION],
    binary = True,
    cmd = "mv $SRC $OUT",
    labels = ["hlink:plz-out/pkg/${OS}_${ARCH}"],
)

filegroup(
    name = "release_files",
    srcs = [
        ":please_tarball",
        ":please_tarball_xz",
    ],
    labels = ["hlink:plz-out/pkg/${OS}_${ARCH}"],
    deps = [
        ":jarcat",
        ":please",
    ],
)

filegroup(
    name = "signer",
    srcs = ["//tools/release_signer"],
    labels = ["hlink:plz-out/pkg"],
)
