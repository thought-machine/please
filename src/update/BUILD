subinclude("//build_defs:version")

go_library(
    name = "update",
    srcs = glob(
        ["*.go"],
        exclude = ["*_test.go"],
    ),
    resources = ["key.pub"],
    visibility = ["PUBLIC"],
    deps = [
        "//src/cli",
        "//src/cli/logging",
        "//src/core",
        "//src/fs",
        "//src/process",
        "//third_party/go:go-crypto",
        "//third_party/go:go-retryablehttp",
        "//third_party/go:semver",
        "//third_party/go:sigstore",
        "//third_party/go:xz",
    ],
)

go_test(
    name = "update_test",
    srcs = [
        "update_test.go",
        "verify_test.go",
    ],
    data = [
        "test_data",
        ":test_please",
        ":test_tarball",
        "//src:please",
    ],
    definitions = {
        "github.com/thought-machine/please/src/core.PleaseVersion": VERSION,
    },
    labels = ["xz"],
    resources = ["key.pub"],
    deps = [
        ":update",
        "//src/cli",
        "//src/cli/logging",
        "//third_party/go:go-retryablehttp",
        "//third_party/go:logging",
        "//third_party/go:testify",
    ],
)

tarball(
    name = "test_tarball",
    srcs = ["//src/update/test:please"],
    out = "please_test.tar.gz",
    subdir = "please",
    test_only = True,
)

genrule(
    name = "test_please",
    srcs = ["//src/update/test:please"],
    out = "please_test",
    binary = True,
    cmd = "cp $SRC $OUT",
    test_only = True,
)
