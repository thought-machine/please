python_binary(
    name = 'plz_build_linter',
    main = 'linter.py',
    resources = ['//src/parse:rule_args'],
    visibility = ['PUBLIC'],
)

python_test(
    name = 'linter_test',
    srcs = ['linter_test.py'],
    data = ['test_data'],
    deps = [
        ':plz_build_linter',
    ],
)
