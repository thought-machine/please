# For testing the go build commands, where we need to be careful about things
# being in subdirectories.
go_library(
    name = "test",
    srcs = ["test.go"],
    test_only = True,
    visibility = ["//test/..."],
)

# Tests a go 'external test', where you can have files in the same directory
# with a _test suffix.
go_test(
    name = "external_test",
    srcs = ["external_test.go"],
    definitions = {
        "test/go_rules/test.Var": "var",
        "test/go_rules/test.Var2": "var1 var2",
    },
    external = True,
    deps = [
        ":test",
        "//third_party/go:testify",
    ],
    definitions = {
        "test/go_rules/test.Var": "var",
        "test/go_rules/test.Var2": "var1 var2",
    },
)
