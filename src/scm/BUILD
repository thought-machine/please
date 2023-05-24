go_library(
    name = "scm",
    srcs = [
        "git.go",
        "scm.go",
        "stub.go",
    ],
    visibility = ["PUBLIC"],
    deps = [
        "///third_party/go/github.com_sourcegraph_go-diff//diff",
        "//src/cli/logging",
        "//src/fs",
    ],
)

go_test(
    name = "git_test",
    srcs = ["git_test.go"],
    data = ["test_data"],
    deps = [
        ":scm",
        "///third_party/go/github.com_stretchr_testify//assert",
    ],
)
