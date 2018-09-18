subinclude("@pleasings//go:go_bindata")

go_bindata(
    name = "embed",
    srcs = ["test.txt"],
    package = "test",  # This argument doesn't exist on the version in our repo.
    prefix = "$PKG",
)

go_test(
    name = "subrepo_subinclude_test",
    srcs = [
        "subrepo_test.go",
        ":embed",
    ],
    deps = ["//third_party/go:testify"],
)
