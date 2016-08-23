deps = []

for src in glob(['*.html'], excludes=['template.html']):
    name = src.replace('.', '_')
    genrule(
        name = name,
	srcs = ['template.html', src],
        tools = [':templater'],
        outs = [src],
        cmd = '$TOOL $SRCS > $OUT',
    )
    deps.append(':' + name)

# Downscale images from original resolution
for colour, scale in [('teal', '50%'), ('white', '20%')]:
    for src in glob(['images/%s*.png' % colour]):
        name = src.replace('.', '_').replace('/', '_')
        genrule(
            name = name,
            srcs = [src],
            outs = [src],
            cmd = 'convert $SRC -crop 500x510+230+30 -resize %s $OUT' % scale,
        )
        deps.append(':' + name)

filegroup(
    name = 'docs',
    srcs = deps + [
        'styles.css',
        'action.js',
        'images/thoughtmachine.png',
        'plz-recording.json',
        ':asciinema-js',
        '//third_party/js:asciinema-css',
    ],
)

# Append our player js onto the end of this one so it runs on load.
genrule(
    name = 'asciinema-js',
    srcs = ['//third_party/js:asciinema-js', 'player.js'],
    outs = ['asciinema-player.js'],
    cmd = 'cat $SRCS > $OUT',
)

go_binary(
    name = 'templater',
    main = 'template.go',
)
