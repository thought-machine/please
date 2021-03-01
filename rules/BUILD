build_defs = glob(["*.build_defs"])

go_library(
    name = "rules",
    resources = build_defs,
    srcs = ["utils.go"],
    visibility = ["PUBLIC"],
)

go_test(
    name = "rules_test",
    srcs = ["rules_test.go"],
    visibility = ["PUBLIC"],
    resources = build_defs,
    deps = [
        ":rules",
        "//third_party/go:testify",
    ],
)
