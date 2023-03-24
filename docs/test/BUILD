go_test(
    name = "docs_test",
    srcs = ["docs_test.go"],
    data = ["//docs"],
    deps = [
        "//src/core",
        "///third_party/go/golang.org_x_net//html",
        "///third_party/go/github.com_stretchr_testify//assert",
        "///third_party/go/github.com_stretchr_testify//require",
    ],
)

# Test that the docs are up-to-date with built-in and plugin rules.
gentest(
    name = "rules_test",
    outs = [
        "lexicon.html",
        "plugins.html",
        "rules.txt",
    ],
    cmd = [
        "for file in docs/*.json; do cat $file | jq '.functions | keys[]' >> rules.txt; done",
        "cp docs/lexicon.html lexicon.html",
        "cp docs/plugins.html plugins.html",
    ],
    labels = ["manual"],
    needs_transitive_deps = True,
    no_test_output = True,
    test_cmd = [
        "numMissing=0",
        "while read line; do if ! grep -q id=$line lexicon.html && ! grep -q id=$line plugins.html; then numMissing=$((numMissing+1)); echo $line \"is missing from the docs\"; fi; done < rules.txt",
        "exit $numMissing",
    ],
    test_tools = [
        "//src:please",
    ],
    deps = [
        "//docs:lexicon_html",
        "//docs:plugins",
        "//docs:plugins_html",
        "//docs:rules",  # built-in rules
    ],
)
