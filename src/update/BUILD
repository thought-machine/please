subinclude("//build_defs:version")

go_library(
    name = "update",
    srcs = [
        "clean.go",
        "link_other.go",
        "link_windows.go",
        "update.go",
        "verify.go",
    ],
    pgo_file = "//:pgo",
    resources = glob(["key.pub"]),
    visibility = ["PUBLIC"],
    deps = [
        "///third_party/go/github.com_coreos_go-semver//semver",
        "///third_party/go/github.com_hashicorp_go-retryablehttp//:go-retryablehttp",
        "///third_party/go/github.com_sigstore_sigstore//pkg/cryptoutils",
        "///third_party/go/github.com_sigstore_sigstore//pkg/signature",
        "///third_party/go/github.com_ulikunitz_xz//:xz",
        "//src/cli",
        "//src/cli/logging",
        "//src/core",
        "//src/fs",
        "//src/process",
        "//src/version",
    ],
)

go_test(
    name = "update_test",
    srcs = [
        "update_test.go",
        "verify_test.go",
    ],
    data = [
        ":test_data",
        ":test_please",
        ":test_tarball",
        "//src:please",
    ],
    definitions = {
        "github.com/thought-machine/please/src/version.PleaseVersion": VERSION,
    },
    labels = ["xz"],
    resources = ["key.pub"],
    # So //test/windows can run this same binary, cross-built, under Wine.
    visibility = ["//test/windows:all"],
    deps = [
        ":update",
        "///third_party/go/github.com_hashicorp_go-retryablehttp//:go-retryablehttp",
        "///third_party/go/github.com_sigstore_sigstore//pkg/cryptoutils",
        "///third_party/go/github.com_sigstore_sigstore//pkg/signature",
        "///third_party/go/github.com_stretchr_testify//assert",
        "///third_party/go/gopkg.in_op_go-logging.v1//:go-logging.v1",
        "//src/cli",
        "//src/core",
        "//src/version",
    ],
)

tarball(
    name = "test_tarball",
    srcs = ["//src/update/test:please"],
    out = "please_test.tar.gz",
    subdir = "please",
    test_only = True,
    # So //test/windows can put it beside the test binary when running it under Wine.
    visibility = ["//test/windows:all"],
)

genrule(
    name = "test_please",
    srcs = ["//src/update/test:please"],
    out = "please_test",
    binary = True,
    cmd = "cp $SRC $OUT",
    test_only = True,
    # So //test/windows can put it beside the test binary when running it under Wine.
    visibility = ["//test/windows:all"],
)

# Exposed so //test/windows can put it beside the test binary when running it under Wine.
filegroup(
    name = "test_data",
    srcs = ["test_data"],
    test_only = True,
    visibility = ["//test/windows:all"],
)
