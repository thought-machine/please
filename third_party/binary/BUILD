GO_CI_LINT_VERSION = "2.13.0"

remote_file(
    name = "golangci-lint",
    binary = True,
    exported_files = ["golangci-lint-%s-${OS}-${ARCH}/golangci-lint" % GO_CI_LINT_VERSION],
    extract = True,
    url = "https://github.com/golangci/golangci-lint/releases/download/v%s/golangci-lint-%s-%s-%s.tar.gz" % (
        GO_CI_LINT_VERSION,
        GO_CI_LINT_VERSION,
        CONFIG.OS,
        CONFIG.ARCH,
    ),
)

PUKU_VERSION = "1.17.1"

remote_file(
    name = "puku",
    binary = True,
    url = f"https://github.com/please-build/puku/releases/download/v{PUKU_VERSION}/puku-{PUKU_VERSION}-{CONFIG.OS}_{CONFIG.ARCH}",
)

# The shell and coreutils that build actions run in on Windows. Windows ships nothing that can
# execute a build action, so Please bundles this rather than requiring an install; see
# docs/design/windows/02-shell-and-build-actions.md.
#
# This is busybox-w64, a third-party fork of busybox (https://frippery.org/busybox/), not an
# upstream busybox release. It is GPL-2.0, so the release notes it separately.
BUSYBOX_VERSION = "FRP-6075-g169694ebd"

remote_file(
    name = "busybox",
    out = "busybox.exe",
    binary = True,
    hashes = ["07bb1e5b095b00d68a695481f9240879f33c5724b40aa2308f999d54ed78f075"],
    licences = ["GPL-2.0"],
    url = f"https://frippery.org/files/busybox/busybox-w64-{BUSYBOX_VERSION}.exe",
    visibility = [
        "//package:all",
        "//test/windows:all",
    ],
)

# A Python for the Wine tests to run .pex files with. This is the embeddable package from
# python.org: python.exe, its DLLs and a zipped standard library, with nothing to install.
#
# It is here rather than as a toolchain because that is all it is for. Nothing Please builds
# depends on it, and no part of the Windows release ships it - a user brings their own Python,
# the same as on any other platform.
PYTHON_WINDOWS_VERSION = "3.11.9"

remote_file(
    name = "python-windows",
    out = "python-windows",
    extract = True,
    hashes = ["009d6bf7e3b2ddca3d784fa09f90fe54336d5b60f0e0f305c37f400bf83cfd3b"],
    licences = ["PSF-2.0"],
    test_only = True,
    url = f"https://www.python.org/ftp/python/{PYTHON_WINDOWS_VERSION}/python-{PYTHON_WINDOWS_VERSION}-embed-amd64.zip",
    visibility = ["//test/windows:all"],
)
