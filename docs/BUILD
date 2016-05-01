deps = []
for src in glob(['*.html'], excludes=['header.html']):
    name = src.replace('.', '_')
    genrule(
        name = name,
	srcs = {
            'SRC': [src],
            'HEADER': ['header.html'],
        },
        tools = [':templater'],
        outs = [src],
        cmd = 'cat $SRCS_SRC | $TOOL $SRCS_HEADER > $OUT',
    )
    deps.append(':' + name)

filegroup(
    name = 'docs',
    srcs = deps + ['skeleton.css'],
)

python_binary(
    name = 'templater',
    main = 'template.py',
)
