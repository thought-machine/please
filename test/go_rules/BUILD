# Test on the builtin go rules

go_library(
    name = 'go_rules_test_lib',
    srcs = ['go_rules_test_lib.go'],
    deps = [
        '//test/go_rules/test',
    ],
    go_tools = [
        '//third_party/go:stringer',
    ],
    test_only = True,
)
go_binary(
    name = 'go_rules_test_bin',
    main = 'go_rules_test_bin.go',
    deps = [
        ':go_rules_test_lib',
        '//test/go_rules/test',
    ],
    test_only = True,
)
go_binary(
    name = 'go_rules_test_generate_bin',
    main = 'go_rules_test_generate_bin.go',
    deps = ['//test/go_rules/test:generate_test'],
    test_only = True,
)

gentest(
    name = 'go_rules_generate_test',
    test_cmd = 'OUT=$($(location :go_rules_test_generate_bin)); if [[ "$OUT" == "Placebo" ]]; then exit 0; else echo "$OUT" && exit 1; fi',
    data = [
        ':go_rules_test_generate_bin',
    ],
    no_test_output = True,
)

gentest(
    name = 'go_rules_test',
    test_cmd = '$(location :go_rules_test_bin)',
    data = [
        ':go_rules_test_bin',
    ],
    no_test_output = True,
)

# Test that doesn't have any actual test cases in it.
go_test(
    name = 'empty_test',
    srcs = ['empty_test.go'],
)
