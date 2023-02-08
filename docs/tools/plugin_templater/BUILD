go_binary(
    name = "plugin_templater",
    srcs = ["main.go"],
    visibility = ["//docs/..."],
    deps = [
        "//third_party/go:cli-init",
        "//docs/tools/lexicon_templater/rules",
        "//docs/tools/plugin_config_tool/plugin:plugin",
    ]
)