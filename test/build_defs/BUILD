package(default_visibility = ["//test/..."])

subinclude("///shell//build_defs:shell", "///go//build_defs:go")

sh_binary(
    name = "content_checker",
    main = "test_file_content.sh",
    visibility = ["//test/..."],
)

filegroup(
    name = "build_defs",
    srcs = ["test.build_defs"],
)

filegroup(
    name = "base_config",
    srcs = [
        ".plzconfig.e2e",
        ".plzconfig_freebsd_amd64",
    ],
)

filegroup(
    name = "cc_embed_binary",
    srcs = ["cc_embed_binary.build_defs"],
    visibility = ["//test/..."],
)

# Used for some of the parse tests that want a subinclude that isn't visible to them.
filegroup(
    name = "not_visible",
    visibility = [],
)

filegroup(
    name = "plugin_go",
    srcs = ["_plugin_go"],
    visibility = ["//test/..."],
)

genrule(
    name = "go_config",
    srcs = ["go_config.template"],
    outs = ["go_config"],
    cmd = "cp $SRC $OUT && sed -i 's|<GO_TOOL>|'$TOOLS'|g' $OUT",
    tools = [CONFIG.GO.GO_TOOL],
)
