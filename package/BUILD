subinclude("//build_defs:version")

filegroup(
    name = "tools",
    srcs = [
        "//tools/build_langserver",
    ] + ([
        # The sandbox is built on Linux namespaces, so there is nothing to ship elsewhere.
        "//tools/sandbox:please_sandbox",
    ] if not is_platform(os = "windows") else []),
    binary = True,
    visibility = ["//src:tools"],
)

filegroup(
    name = "installed_files",
    srcs = [
        ":tools",
        "//src:please",
    ] + ([
        # Windows has no shell that can run a build action, so we ship one. The default
        # [build] shell is 'busybox', which resolves to this once it's installed.
        "//third_party/binary:busybox",
    ] if is_platform(os = "windows") else []),
    binary = True,
    entry_points = {
        "please": "please",
    },
    labels = ["link:plz-out/please"],
    visibility = ["PUBLIC"],
)

# xz only compresses where there is an xz binary to do it, which excludes Windows - the
# busybox we bundle there decompresses only. The gzip tarball is built everywhere, so the
# Windows release is simply the smaller set until it grows a .zip of its own.
XZIP = not is_platform(os = "windows")

if XZIP:
    tarball(
        name = "please_tarball_xz",
        srcs = [":installed_files"],
        out = "please_%s.tar.xz" % VERSION,
        subdir = "please",
        xzip = True,
    )

    tarball(
        name = "please_tools_tarball",
        srcs = [":tools"],
        out = "please_tools_%s.tar.xz" % VERSION,
        subdir = "please_tools",
        xzip = True,
    )

tarball(
    name = "please_tarball",
    srcs = [":installed_files"],
    out = "please_%s.tar.gz" % VERSION,
    subdir = "please",
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
    ] + ([
        ":please_tarball_xz",
        ":please_tools_tarball",
    ] if XZIP else []),
    labels = ["hlink:plz-out/pkg/${OS}_${ARCH}"],
)

filegroup(
    name = "signer",
    srcs = ["//tools/release_signer"],
    labels = ["hlink:plz-out/pkg"],
)
