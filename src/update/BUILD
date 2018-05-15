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
        "//third_party/go:logging",
        "//third_party/go:openpgp",
        "//third_party/go:prompter",
        "//third_party/go:semver",
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
        "//src:please",
    ],
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
    srcs = [
        "//src:please",
    ],
    out = "please_test.tar.gz",
    subdir = "please",
)
