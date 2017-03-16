# Test on the builtin go rules
subinclude('//build_defs:plz_e2e_test')

go_library(
    name = 'go_rules_test_lib',
    srcs = ['go_rules_test_lib.go'],
    go_tools = [
        '//third_party/go:stringer',
    ],
    test_only = True,
    deps = [
        '//test/go_rules/test',
    ],
)

go_binary(
    name = 'go_rules_test_bin',
    srcs = ['go_rules_test_bin.go'],
    test_only = True,
    deps = [
        ':go_rules_test_lib',
        '//test/go_rules/test',
    ],
)

go_binary(
    name = 'go_rules_test_generate_bin',
    srcs = ['go_rules_test_generate_bin.go'],
    test_only = True,
    deps = ['//test/go_rules/test:generate_test'],
)

gentest(
    name = 'go_rules_generate_test',
    data = [
        ':go_rules_test_generate_bin',
    ],
    no_test_output = True,
    test_cmd = 'OUT=$($(location :go_rules_test_generate_bin)); if [[ "$OUT" == "Placebo" ]]; then exit 0; else echo "$OUT" && exit 1; fi',
)

gentest(
    name = 'go_rules_test',
    data = [
        ':go_rules_test_bin',
    ],
    no_test_output = True,
    test_cmd = '$(location :go_rules_test_bin)',
)

# Test that doesn't have any actual test cases in it.
go_test(
    name = 'empty_test',
    srcs = ['empty_test.go'],
)

# Test in which only one case passes
go_test(
    name = 'single_test',
    srcs = ['single_test.go'],
    labels = ['manual'],
)

plz_e2e_test(
    name = 'go_test_spec_test',
    cmd = 'plz test //test/go_rules:single_test Test2',
)
