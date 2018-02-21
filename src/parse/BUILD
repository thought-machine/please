subinclude('//build_defs:engines')

go_library(
    name = 'parse',
    srcs = [
        'init.go',
        'parse_step.go',
        'suggest.go',
    ],
    deps = [
        '//src/core',
        '//src/parse/asp',
        '//src/parse/asp/builtins',
        '//src/utils',
        '//third_party/go:gcfg',
        '//third_party/go:logging',
    ],
    visibility = ['PUBLIC'],
)

filegroup(
    name = 'builtin_rules',
    srcs = glob(['rules/*.build_defs']),
    visibility = ['//src/parse/asp/builtins:all'],
)

# Note that because these scripts are used during the bootstrap process, they can't
# be python_binary rules because we haven't built anything to pex them with yet.
genrule(
    name = 'type_checked_rules',
    srcs = ['type_checking.py'] + glob(['rules/*.build_defs']),
    outs = [basename(x) for x in glob(['rules/*.build_defs'], exclude = [
        'config_rules.build_defs',
        'subrepo_rules.build_defs',
    ])],
    cmd = 'python $SRCS',
)

genrule(
    name = 'rule_args',
    srcs = ['extract_rule_args.py'] + [':type_checked_rules'],
    outs = ['rule_args.json'],
    cmd = 'python $SRCS > $OUT',
    visibility = ['PUBLIC'],
)

go_test(
    name = 'parse_step_test',
    srcs = ['parse_step_test.go'],
    deps = [
        ':parse',
        '//src/core',
        '//third_party/go:testify',
    ],
)

go_test(
    name = 'suggest_test',
    srcs = ['suggest_test.go'],
    deps = [
        ':parse',
        '//third_party/go:testify',
    ],
)

gentest(
    name = 'lint_builtin_rules_test',
    data = glob(['rules/*.build_defs']) + ['//tools/linter'],
    no_test_output = True,
    test_cmd = '$(location //tools/linter) src/parse/rules/*.build_defs',
)
