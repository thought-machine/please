deps = []

for src in glob(['*.html'], exclude=['template.html', 'lexicon*.html', 'language.html']):
    templated_rule = genrule(
        name = src.replace('.', '_'),
        srcs = ['template.html', src],
        tools = [':templater'],
        outs = [src],
        cmd = '$TOOL $SRCS > $OUT',
    )
    deps += [templated_rule]

genrule(
    name = "lexicon_html",
    srcs = [
        "lexicon.html",
        "lexicon_entry.html",
        "template.html",
        "//src/parse/args",
    ],
    outs = ["lexicon.html"],
    cmd = "$TOOLS_LEX > temp.html && $TOOLS_TMPL docs/template.html temp.html > $OUT",
    tools = {
        "lex": [":lexicon_templater"],
        "tmpl": [":templater"],
    },
)

genrule(
    name = "language_html",
    srcs = {
        "html": [
            "template.html",
            "language.html",
        ],
        "grammar": ["grammar.txt"],
    },
    outs = ["language.html"],
    cmd = "$TOOLS_TEMPLATE $SRCS_HTML | $TOOLS_GRAMMAR > $OUT",
    tools = {
        "grammar": [":grammar_templater"],
        "template": [":templater"],
    },
)

# Downscale images from original resolution
for colour, scale in [('teal', '50%'), ('white', '20%')]:
    for src in glob(['images/%s*.png' % colour]):
        image_rule = genrule(
            name = src.replace('.', '_').replace('/', '_'),
            srcs = [src],
            outs = [src],
            cmd = 'mkdir -p images && $TOOL $SRC -crop 500x510+230+30 -resize %s $OUT' % scale,
            tools = ['convert'],
        )
        deps += [image_rule]

filegroup(
    name = "docs",
    srcs = deps + [
        ":language_html",
        ":lexicon_html",
        "styles.css",
        "action.js",
        "images/thoughtmachine.png",
        "plz-recording.json",
        "//third_party/js:asciinema",
    ],
)

go_binary(
    name = "templater",
    srcs = ["template.go"],
)

go_binary(
    name = "lexicon_templater",
    srcs = ["template_lexicon.go"],
)

go_binary(
    name = "grammar_templater",
    srcs = ["template_grammar.go"],
)

# Convenient for popping open the docs in a browser after building (must use 'plz run').
sh_cmd(
    name = "view",
    cmd = "sensible-browser $(out_location :index_html)",
    deps = [
        ":docs",
        ":index_html",
    ],
)
