subinclude('//build_defs:engines')

cgo_library(
    name = 'parse',
    srcs = ['interpreter.go'],
    hdrs = ['interpreter.h'],
    c_srcs = ['interpreter.c'],
    compiler_flags = ['--std=c99 -Werror -DBUILD_STATIC_INTERPRETER'],
    go_srcs = glob(['*.go'], excludes = [
        '*_test.go',
        'builtin_rules.bindata.go',
        'interpreter.go',
    ]) + [':builtin_rules_data'],
    linker_flags = [
        '-Lsrc/parse/cffi -lplease_parser_static',
        PARSER_LINKER_FLAGS,
        '-lm -lz -lutil',
    ],
    deps = [
        '//src/core',
        '//src/parse/asp',
        '//src/parse/asp/builtins',
        '//src/parse/cffi:please_parser_static',
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

go_bindata(
    name = 'builtin_rules_data',
    srcs = [':type_checked_rules'],
)

cgo_test(
    name = 'parse_step_test',
    srcs = ['parse_step_test.go'],
    deps = [
        ':parse',
        '//src/core',
        '//third_party/go:testify',
    ],
)

cgo_test(
    name = 'interpreter_test',
    srcs = ['interpreter_test.go'],
    data = glob([
        'test_data/**/TEST_BUILD',
        'test_data/**/test.py',
    ]),
    deps = [
        ':parse',
        '//third_party/go:testify',
    ],
)

cgo_test(
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
