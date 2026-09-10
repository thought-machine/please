go_library(
    name = "process",
    srcs = [
        "exec_linux.go",
        "exec_other.go",
        "exec_replace_other.go",
        "exec_replace_windows.go",
        "exec_windows.go",
        "kill_other.go",
        "kill_windows.go",
        "output.go",
        "pgroup_other.go",
        "pgroup_windows.go",
        "process.go",
        "progress.go",
        "shell_other.go",
        "shell_windows.go",
    ],
    pgo_file = "//:pgo",
    visibility = ["PUBLIC"],
    deps = [
        "///third_party/go/github.com_peterebden_go-deferred-regex//:go-deferred-regex",
        "//src/cli",
        "//src/cli/logging",
    ] + (["///third_party/go/golang.org_x_sys//windows"] if is_platform(os = "windows") else []),
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
