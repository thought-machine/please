deps = []

for src in glob(['*.html'], excludes=['header.html']):
    name = src.replace('.', '_')
    genrule(
        name = name,
	srcs = {
            'src': [src],
            'header': ['header.html'],
            'templater': ['template.py'],
        },
        outs = [src],
        cmd = 'cat $SRCS_SRC | python $SRCS_TEMPLATER $SRCS_HEADER > $OUT',
    )
    deps.append(':' + name)

filegroup(
    name = 'docs',
    srcs = deps + ['skeleton.css'],
)
