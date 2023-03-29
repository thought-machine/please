go_library(
    name = "process",
    srcs = [
        "exec_linux.go",
        "exec_other.go",
        "output.go",
        "process.go",
        "progress.go",
    ],
    visibility = ["PUBLIC"],
    deps = [
        "///third_party/go/github.com_peterebden_go-deferred-regex//:go-deferred-regex",
        "//src/cli",
        "//src/cli/logging",
    ],
)

go_test(
    name = "process_test",
    srcs = [
        "process_test.go",
        "progress_test.go",
    ],
    deps = [
        ":process",
        "///third_party/go/github.com_stretchr_testify//assert",
    ],
)
