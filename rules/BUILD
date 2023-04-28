build_defs = glob(["*.build_defs"])

go_library(
    name = "rules",
    srcs = ["rules.go"],
    resources = build_defs,
    visibility = ["PUBLIC"],
)

go_test(
    name = "rules_test",
    srcs = ["rules_test.go"],
    resources = build_defs,
    visibility = ["PUBLIC"],
    deps = [
        ":rules",
        "///third_party/go/github.com_stretchr_testify//assert",
    ],
)
