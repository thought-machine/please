go_library(
    name = "update",
    srcs = [
        "clean.go",
        "update.go",
        "verify.go",
        ":pubkey",
    ],
    visibility = ["PUBLIC"],
    deps = [
        "//src/cli",
        "//src/core",
        "//src/process",
        "//third_party/go:logging",
        "//third_party/go:openpgp",
        "//third_party/go:semver",
        "//third_party/go:xz",
    ],
)

go_bindata(
    name = "pubkey",
    srcs = ["pubkey.gpg.asc"],
)

go_test(
    name = "update_test",
    srcs = ["update_test.go"],
    data = [
        ":test_tarball",
        ":test_tarball_xz",
        "//src:please",
    ],
    labels = ["xz"],
    deps = [
        ":update",
        "//src/cli",
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

tarball(
    name = "test_tarball_xz",
    srcs = ["//src/update/test:please"],
    out = "please_test.tar.xz",
    subdir = "please",
    test_only = True,
    xzip = True,
)
