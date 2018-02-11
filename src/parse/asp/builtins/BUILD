genrule(
    name = 'builtin_rule_srcs',
    srcs = ['//src/parse:builtin_rules'],
    outs = [
        # TODO(peterebden): This will be nicer when we can move it alongside these rules
        #                   (so that we can take advantage of globs etc).
        'c_rules.build_defs',
        'cc_rules.build_defs',
        'config_rules.build_defs',
        'go_rules.build_defs',
        'java_rules.build_defs',
        'misc_rules.build_defs',
        'proto_rules.build_defs',
        'python_rules.build_defs',
        'sh_rules.build_defs',
        'subrepo_rules.build_defs',
    ],
    cmd = 'mv $SRCS $TMP_DIR',
)

genrule(
    name = 'builtin_rules',
    srcs = [
        'builtins.build_defs',
        ':builtin_rule_srcs',
    ],
    outs = [
        'builtins.build_defs.gob',
        'c_rules.build_defs.gob',
        'cc_rules.build_defs.gob',
        'config_rules.build_defs.gob',
        'go_rules.build_defs.gob',
        'java_rules.build_defs.gob',
        'misc_rules.build_defs.gob',
        'proto_rules.build_defs.gob',
        'python_rules.build_defs.gob',
        'sh_rules.build_defs.gob',
        'subrepo_rules.build_defs.gob',
    ],
    cmd = '$TOOL -o $TMP_DIR $SRCS',
    tools = ['//src/parse/asp/main:compiler'],
)

go_bindata(
    name = 'builtin_data',
    srcs = [
        ':builtin_rule_srcs',
        ':builtin_rules',
    ],
)

go_library(
    name = 'builtins',
    srcs = [':builtin_data'],
    deps = ['//src/parse/asp'],
    visibility = ['//src/parse/...'],
)
