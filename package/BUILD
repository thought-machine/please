subinclude("//build_defs:version")

# arcat, built for Windows. Please normally downloads a prebuilt arcat keyed by platform and
# there is no Windows one, so the release carries this instead; [build] arcattool defaults to
# it once it is installed beside please.exe. See docs/design/windows/08-offline-release.md.
#
# Renamed because Windows will not run a file whose name has no extension in PATHEXT, and the
# go plugin names a binary after its rule. cp, rather than asking for a different out, because
# the target is not ours to change.
#
# please_go, please_cc and please_pex are not here, and cannot be: cross-compiling a plugin's
# own tool through ///go//tools/please_go and friends collides on subrepo names, because an arch
# subrepo's name does not carry the subrepo that owns it, so the plugin's third_party/go and
# ours both claim third_party/go/github.com_stretchr_testify@windows_amd64. Until those three
# have windows_amd64 releases to download, a native Windows plz can parse and extract plugins
# but cannot build a Go, C++ or Python target. Cross-building from Linux is unaffected, since
# tools resolve to the host.
if is_platform(os = "windows"):
    genrule(
        name = "arcat",
        srcs = ["///third_party/go/github.com_please-build_arcat//:arcat"],
        outs = ["arcat.exe"],
        binary = True,
        cmd = "cp $SRC $OUT",
    )

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
        # Windows has no installer and no package manager to carry the instructions, so they
        # travel in the zip.
        ":install_md",
        # Extracting a plugin needs arcat, and there is no arcat release for Windows, so a
        # fresh install there cannot get a plugin at all without one of its own.
        ":arcat",
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

filegroup(
    name = "install_md",
    srcs = ["Install.md"],
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
        # //test/windows runs the artifact itself, rather than a reconstruction of it.
        visibility = ["//test/windows:all"],
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
