subinclude("//build_defs:version")

go_library(
    name = "update",
    srcs = [
        "clean.go",
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
        "test_data",
        ":test_please",
        ":test_tarball",
        "//src:please",
    ],
    definitions = {
        "github.com/thought-machine/please/src/version.PleaseVersion": VERSION,
    },
    labels = ["xz"],
    resources = ["key.pub"],
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
)

genrule(
    name = "test_please",
    srcs = ["//src/update/test:please"],
    out = "please_test",
    binary = True,
    cmd = "cp $SRC $OUT",
    test_only = True,
)
