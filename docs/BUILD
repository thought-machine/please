subinclude("//docs/build_defs")

html_template(
    name = "acknowledgements_html",
    src = "acknowledgements.html",
    title = "Acknowledgements",
)

html_template(
    name = "basics_html",
    src = "basics.html",
    title = "Please basics",
)

html_template(
    name = "build_rules_html",
    src = "build_rules.html",
    title = "Writing additional build rules",
)

html_template(
    name = "cache_html",
    src = "cache.html",
    title = "Please caching system",
)

html_template(
    name = "codelabs_html",
    src = "//docs/codelabs:templated_index",
    output = "codelabs.html",
    title = "Codelabs",
)

html_template(
    name = "commands_html",
    src = "commands.html",
    title = "Please commands",
)

genrule(
    name = "config_content",
    srcs = [
        "config.html",
    ],
    outs = ["config_content.html"],
    cmd = '"$TOOL" --template "$SRC" > "$OUT"',
    tools = ["//docs/tools/config_templater"],
)

html_template(
    name = "config_html",
    src = ":config_content",
    output = "config.html",
    title = "Please config file reference",
)

html_template(
    name = "cross_compiling_html",
    src = "cross_compiling.html",
    title = "Cross-compiling",
)

html_template(
    name = "dependencies_html",
    src = "dependencies.html",
    title = "Third-party dependencies",
)

html_template(
    name = "faq_html",
    src = "faq.html",
    title = "Please FAQ",
)

html_template(
    name = "index_html",
    src = "index.html",
    title = "Please",
)

genrule(
    name = "language_content",
    srcs = {
        "html": "language.html",
        "grammar": "grammar.txt",
    },
    outs = ["language_content.html"],
    cmd = '"$TOOL" --in "$SRCS_HTML" --grammar "$SRCS_GRAMMAR" > "$OUT"',
    tools = ["//docs/tools/grammar_templater"],
)

html_template(
    name = "language_html",
    src = ":language_content",
    output = "language.html",
    title = "The Please BUILD language",
)

genrule(
    name = "rules",
    outs = ["rules.json"],
    cmd = '"$TOOL" init && "$TOOL" query rules > "$OUT"',
    local = True,
    sandbox = False,
    tools = ["//src:please"],
    visibility = ["//docs/test/..."],
)

genrule(
    name = "lexicon_content",
    srcs = {
        "lexicon": "lexicon.html",
        "lexicon_entry": "lexicon_entry.html",
        "rules": ":rules",
    },
    outs = ["lexicon_content.html"],
    cmd = '"$TOOL" -i "$SRCS_LEXICON" -i "$SRCS_LEXICON_ENTRY" "$SRCS_RULES" > "$OUT"',
    tools = ["//docs/tools/lexicon_templater"],
    visibility = [":lexicon_html"],
)

html_template(
    name = "lexicon_html",
    src = ":lexicon_content",
    output = "lexicon.html",
    title = "Please Lexicon",
)

html_template(
    name = "performance_html",
    src = "performance.html",
    title = "Performance report",
)

html_template(
    name = "pleasings_html",
    src = "pleasings.html",
    title = "Extra rules (aka. Pleasings)",
)

# Plugin versions to pull the docs from
plugins = {
    "python": "v1.7.0",
    "java": "v0.4.0",
    "go": "v1.17.4",
    "cc": "v0.4.0",
    "shell": "v0.2.0",
    "go-proto": "v0.3.0",
    "python-proto": "v0.1.0",
    "proto": "v0.4.0",
}

plugin_targets = []

for plugin, revision in plugins.items():
    plugin_rules_target = plugin_rules(name = plugin, revision = revision)
    plugin_targets += [plugin_rules_target]

filegroup(
    name = "plugins",
    exported_deps = plugin_targets,
    visibility = ["//docs/test/..."],
)

genrule(
    name = "plugins_content",
    srcs = {
        "lexicon_entry": "lexicon_entry.html",
        "plugins": "plugins.html",
        "plugin_data": plugin_targets,
    },
    outs = ["plugins_content.html"],
    cmd = [
        '"$TOOL" --plugin $SRCS_PLUGINS --lex $SRCS_LEXICON_ENTRY $SRCS_PLUGIN_DATA > "$OUT"',
    ],
    tools = ["//docs/tools/plugin_templater"],
    visibility = [":plugins_html"],
)

html_template(
    name = "plugins_html",
    src = ":plugins_content",
    output = "plugins.html",
    title = "Please plugins",
)

html_template(
    name = "post_build_html",
    src = "post_build.html",
    title = "Pre- and post-build functions",
)

html_template(
    name = "quickstart_html",
    src = "quickstart.html",
    title = "Please quickstart",
)

html_template(
    name = "quickstart_dropoff_html",
    src = "quickstart_dropoff.html",
    title = "What's next?",
)

html_template(
    name = "remote_builds_html",
    src = "remote_builds.html",
    title = "Remote build execution",
)

html_template(
    name = "require_provide_html",
    src = "require_provide.html",
    title = "Require & Provide",
)

html_template(
    name = "tests_html",
    src = "tests.html",
    title = "Testing with Please",
)

for src in glob(["milestones/*.html"]):
    version = src.removeprefix("milestones/").removesuffix(".html")
    html_template(
        name = src.replace(".", "_").replace("/", "_"),
        src = src,
        title = f"Please v{version}",
    )

filegroup(
    name = "docs",
    srcs = glob(["images/*.png"]) + [
        # Pages
        ":acknowledgements_html",
        ":basics_html",
        ":build_rules_html",
        ":cache_html",
        ":codelabs_html",
        ":commands_html",
        ":config_html",
        ":cross_compiling_html",
        ":dependencies_html",
        ":faq_html",
        ":index_html",
        ":language_html",
        ":lexicon_html",
        ":performance_html",
        ":pleasings_html",
        ":plugins_html",
        ":post_build_html",
        ":quickstart_html",
        ":quickstart_dropoff_html",
        ":remote_builds_html",
        ":require_provide_html",
        ":tests_html",
        "error.html",

        # Milestone pages
        ":milestones_15_9_1_html",
        ":milestones_16_0_0_html",
        ":milestones_17_0_0_html",

        # CSS
        "tachyons-4.12.0.min.css",
        "fonts.css",
        "styles.css",

        # JS
        "action.js",
        "tabs.js",

        # Images
        "images/thoughtmachine.png",
        "favicon",

        # Media
        "plz-recording.json",
        "//third_party/js:asciinema",
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
