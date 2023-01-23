subinclude("//docs/build_defs")

deps = []

for src in glob(
    [
        "*.html",
        "milestones/*.html",
    ],
    exclude = [
        "template.html",
        "lexicon*.html",
        "language.html",
        "config.html",
        "error.html",
        "plugins.html",
    ],
):
    deps += [html_template(name = src.replace(".", "_").replace("/", "_"), src = src)]

html_template(
    name = "codelabs_html",
    src = "//docs/codelabs:templated_index",
    output = "codelabs.html",
)

genrule(
    name = "rules",
    outs = ["rules.json"],
    cmd = '"$TOOL" init && "$TOOL" query rules > "$OUT"',
    local = True,
    sandbox = False,
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
        '"$TOOLS_LEX" -i docs/lexicon.html -i docs/lexicon_entry.html -r docs/rules.json > "$OUT"',
        '"$TOOLS_TMPL" --template docs/template.html --in lexicon.html > tmp.html',
        'mv tmp.html "$OUT"',
    ],
    tools = {
        "lex": ["//docs/tools/lexicon_templater"],
        "tmpl": ["//docs/tools/templater"],
    },
)

plugins = {
    "python": "v1.2.0",
    "java": "v0.2.5",
    "go": "v1.1.4",
    "cc": "v0.3.2",
    "shell": "v0.1.1",
}

plugin_lex_cmd = "\"$TOOLS_LEX\" -i docs/plugins.html -i docs/lexicon_entry.html"
for plugin, revision in plugins.items():
    plugin_rules(
        name = plugin,
        revision = revision,
    )

    plugin_lex_cmd += " -r docs/%s_rules.json" % plugin

genrule(
    name = "plugins_html",
    srcs = deps + [
        "lexicon.html",
        "lexicon_entry.html",
        "template.html",
        "plugins.html",
        ":python",
        ":java",
        ":go",
        ":cc",
        ":shell",
    ],
    outs = ["plugins.html"],
    cmd = [
        plugin_lex_cmd + " > \"$OUT\"",
        '"$TOOLS_TMPL" --template docs/template.html --in plugins.html > tmp.html',
        'mv tmp.html "$OUT"',
    ],
    tools = {
        "lex": ["//docs/tools/lexicon_templater"],
        "tmpl": ["//docs/tools/templater"],
    },
)

genrule(
    name = "config_html",
    srcs = [
        "config.html",
        "template.html",
    ],
    outs = ["config.html"],
    cmd = [
        '"$TOOLS_CONFIG" > config.html',
        '"$TOOLS_TMPL" --template docs/template.html --in config.html > tmp.html',
        'mv tmp.html "$OUT"',
    ],
    tools = {
        "config": ["//docs/tools/config_templater"],
        "tmpl": ["//docs/tools/templater"],
    },
)

genrule(
    name = "language_html",
    srcs = {
        "template": ["template.html"],
        "html": ["language.html"],
        "grammar": ["grammar.txt"],
    },
    outs = ["language.html"],
    cmd = '"$TOOLS_TEMPLATE" --template $SRCS_TEMPLATE --in $SRCS_HTML | "$TOOLS_GRAMMAR" > "$OUT"',
    tools = {
        "grammar": ["//docs/tools/grammar_templater"],
        "template": ["//docs/tools/templater"],
    },
)

filegroup(
    name = "docs",
    srcs = glob(["images/*.png"]) + deps + [
        ":language_html",
        ":plugins_html",
        ":lexicon_html",
        ":config_html",
        "tachyons-4.12.0.min.css",
        "fonts.css",
        "styles.css",
        "action.js",
        "tabs.js",
        "error.html",
        "images/thoughtmachine.png",
        "plz-recording.json",
        "//third_party/js:asciinema",
        "favicon",
        ":codelabs_html",
    ],
    visibility = ["//docs/..."],
    deps = ["//docs/codelabs"],
)

#TODO(jpoole): unify these two once we switch to the new website
tarball(
    name = "tarball",
    srcs = [":docs"],
    out = "docs.tar.gz",
    labels = ["hlink:plz-out/pkg"],
)

# This is used exclusively for the s3 website
tarball(
    name = "deep-tarball",
    srcs = [
        ":docs",
        "//docs/codelabs",
    ],
    out = "deep-docs.tar.gz",
    flatten = False,
    labels = ["hlink:plz-out/pkg"],
)

filegroup(
    name = "template_html",
    srcs = ["template.html"],
    visibility = ["//docs/..."],
)

results_files = [
    "all_results",
    "build_target_benchmark_all_results",
]

genrule(
    name = "performance_data",
    outs = [f"performance/{file}.jsonl" for file in results_files],
    cmd = [f"curl https://please.build/performance/{file}.jsonl > performance/{file}.jsonl" for file in results_files],
)

# Convenient for popping open the docs in a browser after building (must use 'plz run').
sh_cmd(
    name = "view",
    cmd = "sensible-browser http://localhost:8080 && cd plz-out/gen/docs && python3 -m http.server 8080",
    data = [
        ":docs",
        ":performance_data",
    ],
)

sh_cmd(
    name = "server",
    cmd = "cd plz-out/gen/docs && python3 -m http.server 8080",
    data = [
        ":docs",
        ":performance_data",
    ],
)
