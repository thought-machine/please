go_binary(
    name = 'cache_cleaner',
    main = ':cache_cleaner_platform',
    deps = [
        '//src/output',
        '//third_party/go:logging',
        '//third_party/go:humanize',
    ],
    visibility = ['PUBLIC'],
)

# Bit of a fiddle to do conditional compilation (weirdly, Go's isn't working for me).
genrule(
    name = 'cache_cleaner_platform',
    srcs = ['cache_cleaner.go'],
    outs = ['cache_cleaner_platform.go'],
    cmd = ('cat $SRC | sed -e "s/stat.Atim.Sec/stat.Atimespec.Sec/g" > $OUT'
           if CONFIG.OS == 'darwin' else 'cp $SRC $OUT')
)

go_test(
    name = 'cache_cleaner_test',
    srcs = ['cache_cleaner_test.go', ':cache_cleaner_platform'],
    deps = [
        '//src/output',
        '//third_party/go:logging',
        '//third_party/go:humanize',
        '//third_party/go:testify',
    ],
)
