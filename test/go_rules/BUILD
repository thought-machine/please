# Test on the builtin go rules
subinclude("//build_defs:plz_e2e_test")

go_library(
    name = "go_rules_test_lib",
    srcs = ["go_rules_test_lib.go"],
    test_only = True,
    deps = [
        "//test/go_rules/test",
    ],
)

go_binary(
    name = "go_rules_test_bin",
    srcs = ["go_rules_test_bin.go"],
    test_only = True,
    deps = [
        ":go_rules_test_lib",
        "//test/go_rules/test",
    ],
)

gentest(
    name = "go_rules_test",
    data = [
        ":go_rules_test_bin",
    ],
    no_test_output = True,
    test_cmd = "$(location :go_rules_test_bin)",
)

# Test that doesn't have any actual test cases in it.
go_test(
    name = "empty_test",
    srcs = ["empty_test.go"],
)

# Test in which only one case passes
go_test(
    name = "single_test",
    srcs = ["single_test.go"],
    labels = ["manual"],
)

plz_e2e_test(
    name = "go_test_spec_test",
    cmd = "plz test //test/go_rules:single_test Test2",
)

go_test(
    name = "example_test",
    srcs = ["example_test.go"],
)

go_library(
    name = "test_main_lib",
    srcs = ["main_test.go"],
    out = "whatever.a",
    test_only = True,
)

go_test_main(
    name = "test_main",
    srcs = ["main_test.go"],
    test_only = True,
)

go_binary(
    name = "test_main_bin",
    srcs = [":test_main"],
    test_only = True,
    deps = [":test_main_lib"],
)

gentest(
    name = "test_main_test",
    data = [":test_main_bin"],
    test_cmd = "$DATA | tee $RESULTS_FILE",
)

go_library(
    name = "package_test_lib",
    srcs = ["package_lib.go"],
    package = "package_lib",
)

go_test(
    name = "package_test",
    srcs = ["package_test.go"],
    external = True,
    deps = [":package_test_lib"],
)
