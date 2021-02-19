go_library(
    name = "update",
    srcs = glob(
        ["*.go"],
        exclude = ["*_test.go"],
    ),
    visibility = ["PUBLIC"],
    deps = [
        "//src/cli",
        "//src/core",
        "//src/fs",
        "//src/process",
        "//src/utils",
        "//third_party/go:go-retryablehttp",
        "//third_party/go:logging",
        "//third_party/go:semver",
        "//third_party/go:xcrypto",
        "//third_party/go:xz",
    ],
)

go_test(
    name = "update_test",
    srcs = ["update_test.go"],
    data = [
        ":test_tarball",
        ":test_please",
        "//src:please",
    ],
    labels = ["xz"],
    deps = [
        ":update",
        "//src/cli",
        "//third_party/go:go-retryablehttp",
        "//third_party/go:logging",
        "//third_party/go:testify",
    ],
)

go_test(
    name = "verify_test",
    srcs = ["verify_test.go"],
    data = ["test_data"],
    deps = [
        ":update",
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
    cmd = "cp $SRC $OUT",
    out = "please_test",
    binary = True,
    test_only = True,
)
