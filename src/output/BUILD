subinclude("//build_defs:ansi_escapes")

replace_ansi_escapes(
    name = "ansi_replacements",
    srcs = [
        "interactive_display.go",
        "shell_output.go",
    ],
)

go_library(
    name = "output",
    srcs = [
        "print.go",
        "trace.go",
        ":ansi_replacements",
    ],
    visibility = ["PUBLIC"],
    deps = [
        "//src/build",
        "//src/cli",
        "//src/core",
        "//src/test",
        "//third_party/go:go-flags",
        "//third_party/go:humanize",
        "//third_party/go:logging",
        "//third_party/go:terminal",
    ],
)

go_test(
    name = "interactive_display_test",
    srcs = ["interactive_display_test.go"],
    deps = [
        ":output",
        "//third_party/go:testify",
    ],
)

go_test(
    name = "shell_output_test",
    srcs = ["shell_output_test.go"],
    deps = [
        ":output",
        "//src/core",
        "//third_party/go:testify",
    ],
)
