subinclude("//build_defs:version")

filegroup(
    name = "tools",
    srcs = [
        "//tools/build_langserver",
        "//tools/java:javac_worker",
        "//tools/java:junit_runner",
        "//tools/please_go",
        "//tools/please_go_embed",
        "//tools/please_go_filter",
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
    entry_points = {
        "please": "please",
    },
    labels = ["link:plz-out/please"],
    visibility = ["PUBLIC"],
)

tarball(
    name = "please_tarball_xz",
    srcs = [":installed_files"],
    out = "please_%s.tar.xz" % VERSION,
    subdir = "please",
    xzip = True,
)

tarball(
    name = "please_tarball",
    srcs = [":installed_files"],
    out = "please_%s.tar.gz" % VERSION,
    subdir = "please",
)

tarball(
    name = "please_tools_tarball",
    srcs = [":tools"],
    out = "please_tools_%s.tar.xz" % VERSION,
    subdir = "please_tools",
    xzip = True,
)

genrule(
    name = "please",
    srcs = ["//src:please"],
    outs = ["please_" + VERSION],
    binary = True,
    cmd = "cp $SRC $OUT",
)

genrule(
    name = "please_shim",
    srcs = ["//tools/please_shim"],
    outs = ["please_shim_" + VERSION],
    binary = True,
    cmd = "cp $SRC $OUT",
)

filegroup(
    name = "release_files",
    srcs = [
        ":please",
        ":please_shim",
        ":please_tarball",
        ":please_tarball_xz",
        ":please_tools_tarball",
    ],
    labels = ["hlink:plz-out/pkg/${OS}_${ARCH}"],
)

filegroup(
    name = "signer",
    srcs = ["//tools/release_signer"],
    labels = ["hlink:plz-out/pkg"],
)
