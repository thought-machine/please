go_library(
    name = "help",
    srcs = [
        "completion.go",
        "config.go",
        "help.go",
        "misc_topics.go",
        "rules.go",
    ],
    pgo_file = "//:pgo",
    visibility = ["PUBLIC"],
    deps = [
        "///third_party/go/github.com_peterebden_go-deferred-regex//:go-deferred-regex",
        "///third_party/go/github.com_thought-machine_go-flags//:go-flags",
        "///third_party/go/golang.org_x_exp//maps",
        "//rules",
        "//src/cli",
        "//src/cli/logging",
        "//src/core",
        "//src/fs",
        "//src/parse/asp",
        "//src/plz",
        "//src/version",
    ],
)

go_test(
    name = "help_test",
    srcs = [
        "help_test.go",
        "rules_test.go",
    ],
    deps = [
        ":help",
        "///third_party/go/github.com_stretchr_testify//assert",
        "//src/core",
    ],
)
