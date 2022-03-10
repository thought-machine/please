subinclude("//test/build_defs:build_defs")

go_binary(
    name = "main",
    srcs = ["main.go"],
    definitions = {"github.com/thought-machine/please/test/go_rules/definitions/version.Version": "v1.0.0"},
    deps = ["//test/go_rules/definitions/version"],
)

plz_e2e_test(
    name = "definitions_test",
    cmd = "plz run //test/go_rules/definitions:main",
    expect_output_contains = "v1.0.0",
)
