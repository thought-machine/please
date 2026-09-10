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
        # install.sh gets the short name with 'ln -sf please plz'. Symlinks need Developer
        # Mode on Windows, so a one-line batch file stands in for it.
        ":plz_cmd",
    ] if is_platform(os = "windows") else []),
    binary = True,
    entry_points = {
        "please": "please.exe" if is_platform(os = "windows") else "please",
    },
    labels = ["link:plz-out/please"],
    visibility = ["PUBLIC"],
)

filegroup(
    name = "plz_cmd",
    srcs = ["plz.cmd"],
    binary = True,
)

# xz only compresses where there is an xz binary to do it, which excludes Windows - the
# busybox we bundle there decompresses only. Windows gets a .zip in place of the two xz
# tarballs; the gzip one is built everywhere.
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

# Windows has no guaranteed tar, so its release is a zip. This is built on the Linux release
# box like everything else, so it uses the host's arcat, not a Windows one.
if is_platform(os = "windows"):
    genrule(
        name = "please_zip",
        srcs = [":installed_files"],
        outs = ["please_%s.zip" % VERSION],
        # --rename_dir, rather than --prefix, because entries come out under the package
        # directory they were built in and the release wants them at the top of a please/.
        cmd = "$TOOL zip --dumb --input package --output $OUT --rename_dir package:please",
        tools = [CONFIG.ARCAT_TOOL],
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
    ] if XZIP else [":please_zip"]),
    labels = ["hlink:plz-out/pkg/${OS}_${ARCH}"],
)

filegroup(
    name = "signer",
    srcs = ["//tools/release_signer"],
    labels = ["hlink:plz-out/pkg"],
)
