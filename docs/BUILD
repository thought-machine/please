deps = []
for src in glob(['*.html'], excludes=['header.html']):
    name = src.replace('.', '_')
    genrule(
        name = name,
	srcs = {
            'SRC': [src],
            'HEADER': ['header.html'],
        },
        outs = [src],
        cmd = 'CONTENT=$(sed \'s:/:\\/:g\' $SRCS_HEADER) && sed -ie "s/<!-- HEADER -->/$CONTENT/g" $SRCS_SRC',
    )
    deps.append(':' + name)

filegroup(
    name = 'docs',
    srcs = deps,
)
