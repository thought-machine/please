go_test(
    name = "docs_test",
    srcs = ["docs_test.go"],
    data = ["//docs"],
    deps = [
        "//src/core",
        "//third_party/go:net",
        "//third_party/go:testify",
    ],
)

# Test that the docs are up-to-date with built-in and plugin rules.
gentest(
    name = "rules_test",
    deps = [
        "//docs:plugins",
        "//docs:rules", # built-in rules
        "//docs:lexicon_html",
        "//docs:plugins_html",
    ],
    cmd = [
        "for file in docs/*.json; do cat $file | jq '.functions | keys[]' >> rules.txt; done",
        "cp docs/lexicon.html lexicon.html",
        "cp docs/plugins.html plugins.html",
    ],
    outs = [
        "rules.txt",
        "lexicon.html",
        "plugins.html",
    ],
    test_cmd = [
        "numMissing=0",
        "while read line; do if ! grep -q id=$line lexicon.html && ! grep -q id=$line plugins.html; then numMissing=$((numMissing+1)); echo $line \"is missing from the docs\"; fi; done < rules.txt",
        "exit $numMissing",
    ],
    test_tools = [
        "//src:please",
    ],
    needs_transitive_deps = True,
    no_test_output = True,
    labels = ["manual"],
)
