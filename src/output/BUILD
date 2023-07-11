go_library(
    name = "output",
    srcs = [
        "interactive_display.go",
        "print.go",
        "shell_output.go",
        "targets.go",
        "trace.go",
    ],
    pgo_file = "//:pgo",
    visibility = ["PUBLIC"],
    deps = [
        "///third_party/go/github.com_dustin_go-humanize//:go-humanize",
        "///third_party/go/github.com_peterebden_go-deferred-regex//:go-deferred-regex",
        "//src/cli",
        "//src/cli/logging",
        "//src/core",
        "//src/process",
        "//src/test",
    ],
)

go_test(
    name = "output_test",
    srcs = [
        "interactive_display_test.go",
        "shell_output_test.go",
    ],
    deps = [
        ":output",
        "///third_party/go/github.com_stretchr_testify//assert",
    ],
)
