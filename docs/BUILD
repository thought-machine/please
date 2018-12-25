deps = []

for src in glob(
    ["*.html"],
    exclude = [
        "template.html",
        "lexicon*.html",
        "language.html",
    ],
):
    templated_rule = genrule(
        name = src.replace(".", "_"),
        srcs = ["template.html", src],
        outs = [src],
        cmd = "$TOOL $SRCS > $OUT",
        tools = [":templater"],
    )
    deps += [templated_rule]

genrule(
    name = "rules",
    outs = ["rules.json"],
    cmd = "$TOOL query rules > $OUT",
    tools = ["//src:please"],
)

genrule(
    name = "lexicon_html",
    srcs = deps + [
        "lexicon.html",
        "lexicon_entry.html",
        "template.html",
        ":rules",
    ],
    outs = ["lexicon.html"],
    cmd = [
        "$TOOLS_LEX > lexicon.html",
        "$TOOLS_TMPL docs/template.html lexicon.html > tmp.html",
        "mv tmp.html $OUT",
    ],
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

filegroup(
    name = "docs",
    srcs = glob(["images/*.png"]) + deps + [
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
