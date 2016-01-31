go_library(
    name = 'buildgo',
    srcs = ['find_cover_vars.go'],
    deps = [
        '//src/output',
        '//third_party/go:logging',
    ],
)

go_binary(
    name = 'plz_go_test',
    main = 'plz_go_test.go',
    deps = [
        ':buildgo',
        '//third_party/go:logging',
    ],
    visibility = ['PUBLIC'],
)

go_test(
    name = 'find_cover_vars_test',
    srcs = ['find_cover_vars_test.go'],
    deps = [
        ':buildgo',
        '//third_party/go:testify',
    ],
    data = [
        'test_data/core.a',
        ':test_excluded_archive',
    ],
)

# Little tweak so we don't have to commit this file twice.
genrule(
    name = 'test_excluded_archive',
    srcs = ['test_data/core.a'],
    outs = ['test_data/x/core.a'],
    cmd = 'cp $SRC $OUT',
)
