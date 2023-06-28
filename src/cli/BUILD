go_library(
    name = "cli",
    srcs = [
        "flags.go",
        "logging.go",
        "process.go",
        "progress.go",
        "prompt.go",
        "replacements.go",
        "suggest.go",
        "winch_other.go",
        "winch_windows.go",
        "window.go",
    ],
    visibility = ["PUBLIC"],
    deps = [
        "///third_party/go/github.com_coreos_go-semver//semver",
        "///third_party/go/github.com_dustin_go-humanize//:go-humanize",
        "///third_party/go/github.com_manifoldco_promptui//:promptui",
        "///third_party/go/github.com_peterebden_go-cli-init_v5//flags",
        "///third_party/go/github.com_peterebden_go-cli-init_v5//logging",
        "///third_party/go/github.com_peterebden_go-deferred-regex//:go-deferred-regex",
        "///third_party/go/github.com_texttheater_golang-levenshtein//levenshtein",
        "///third_party/go/github.com_thought-machine_go-flags//:go-flags",
        "///third_party/go/golang.org_x_term//:term",
        "///third_party/go/gopkg.in_op_go-logging.v1//:go-logging.v1",
        "//src/cli/logging",
    ],
)

go_test(
    name = "cli_test",
    srcs = [
        "flags_test.go",
        "logging_test.go",
    ],
    deps = [
        ":cli",
        "///third_party/go/github.com_stretchr_testify//assert",
        "///third_party/go/gopkg.in_op_go-logging.v1//:go-logging.v1",
    ],
)
