# These are the shell command code replacements made in shell_output.go
# Backslashes are deliberately repeated many times to get through all the levels of escaping needed.
REPLACEMENTS = {
    'BOLD': r'\\\\x1b[1m',
    'BOLD_GREY': r'\\\\x1b[30;1m',
    'BOLD_RED': r'\\\\x1b[31;1m',
    'BOLD_GREEN': r'\\\\x1b[32;1m',
    'BOLD_YELLOW': r'\\\\x1b[33;1m',
    'BOLD_BLUE': r'\\\\x1b[34;1m',
    'BOLD_MAGENTA': r'\\\\x1b[35;1m',
    'BOLD_CYAN': r'\\\\x1b[36;1m',
    'BOLD_WHITE': r'\\\\x1b[37;1m',
    'GREY': r'\\\\x1b[30m',
    'RED': r'\\\\x1b[31m',
    'GREEN': r'\\\\x1b[32m',
    'YELLOW': r'\\\\x1b[33m',
    'BLUE': r'\\\\x1b[34m',
    'MAGENTA': r'\\\\x1b[35m',
    'CYAN': r'\\\\x1b[36m',
    'WHITE': r'\\\\x1b[37m',
    'WHITE_ON_RED': r'\\\\x1b[37;41;1m',
    'RED_NO_BG': r'\\\\x1b[31;49;1m',
    'RESET': r'\\\\x1b[0m',
    'ERASE_AFTER': r'\\\\x1b[K',
}

TEMPLATED_FILES = {
    'shell_output.go': 'shell_output_templated',
    'interactive_display.go': 'interactive_display_templated',
}
TEMPLATED_FILES_DEPS = [':' + x for x in sorted(TEMPLATED_FILES.values())]

for file_name, rule_name in sorted(TEMPLATED_FILES.items()):
    genrule(
        name = rule_name,
        srcs = [file_name],
        outs = [rule_name + '.go'],
        cmd = 'cat $SRC | %s > $OUT' % ' | '.join(r'sed -e "s/\\${%s}/%s/g"' % (k, v) for k, v in sorted(REPLACEMENTS.items()))
    )

go_library(
    name = 'output',
    srcs = glob(['*.go'], excludes=TEMPLATED_FILES.keys() + ['*_test.go']) + TEMPLATED_FILES_DEPS,
    deps = TEMPLATED_FILES_DEPS + [
        '//src/build',
        '//src/core',
	'//third_party/go:go-flags',
	'//third_party/go:humanize',
	'//third_party/go:logging',
	'//third_party/go:terminal',
    ],
    visibility = ['PUBLIC'],
)

go_test(
    name = 'interactive_display_test',
    srcs = ['interactive_display_test.go'],
    deps = [
        ':output',
    ],
)

go_test(
    name = 'shell_output_test',
    srcs = ['shell_output_test.go'],
    deps = [
        ':output',
        '//src/core',
    ],
)
