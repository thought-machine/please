package(python_module_dir = '')

python_binary(
    name = 'linter',
    out = 'linter',
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
        ':linter',
    ],
)
