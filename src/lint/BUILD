python_binary(
    name = 'please_build_linter',
    out = 'please_build_linter',
    main = 'linter.py',
    resources = ['//src/parse:rule_args'],
    shebang = '/usr/bin/env python',
    visibility = ['PUBLIC'],
)

python_test(
    name = 'linter_test',
    srcs = ['linter_test.py'],
    data = ['test_data'],
    deps = [
        ':please_build_linter',
    ],
)
