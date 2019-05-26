go_library(
    name = "scm",
    srcs = glob(
        ["*.go"],
        exclude = ["*_test.go"],
    ),
    visibility = ["PUBLIC"],
    deps = [
        "//src/fs",
        "//third_party/go:diff",
        "//third_party/go:logging",
    ],
)

go_test(
    name = "git_test",
    srcs = ["git_test.go"],
    data = ["test_data"],
    deps = [
        ":scm",
        "//third_party/go:testify",
    ],
)
