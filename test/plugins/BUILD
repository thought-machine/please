subinclude("//test/build_defs:build_defs")

genrule(
    name = "foo_plugin",
    srcs = ["foo_plugin"],
    cmd = "mv $SRC foo_plugin && tar -czf $OUT foo_plugin",
    outs = ["foo_plugin.tar.gz"]
)

def plugin_e2e_test(name, plz_command, expected_output=None, expected_failure=False, expect_output_contains=None):
    return please_repo_e2e_test(
        name = name,
        plz_command = plz_command.replace("plz ", "plz -o buildconfig.nested-subrepo:$TMP_DIR/$DATA_NESTED_SUBREPO -o please.pluginrepo:file://$TMP_DIR/$DATA_PLUGIN "),
        repo = "test_repo",
        data = {
            "plugin": [":foo_plugin"],
            "nested_subrepo": ["nested_subrepo"],
        },
        expected_output = expected_output,
        expected_failure = expected_failure,
        expect_output_contains = expect_output_contains,
    )

plugin_e2e_test(
    name = "plugin_test",
    plz_command = "plz build -o plugin.foo.modulepath:something //test:root_lib",
)

plugin_e2e_test(
    name = "cross_compile_test",
    plz_command = "plz build -o plugin.foo.modulepath:something ///linux_arm64//test:root_lib",
)

plugin_e2e_test(
    name = "nested_test",
    expected_output = {
        "plz-out/gen/foo/test/foo_lib.foo": "sha1 88f20b91c6af94a7974d7b547ec49529138a98f2 foolang/test.foo",
        "plz-out/gen/nested/test/nested_lib.foo": "sha256 3e9e58ebd72c21509355abe528401ae4b37f632dc9446de9c1896235fa158856 nested/test.foo",
    },
    plz_command = "plz build -o plugin.foo.modulepath:something ///nested//test:nested_lib ///foo//test:foo_lib",
)

plugin_e2e_test(
    name = "plugin_test_non_optional_config",
    expected_failure = True,
    plz_command = "plz build //test:root_lib",
)

plugin_e2e_test(
    name = "override_test",
    expected_output = {
        "plz-out/gen/override/fooc": "tool: overriden, path: something, flags: [compileflagoverride]",
    },
    plz_command = "plz build -o plugin.foo.modulepath:something -o plugin.foo.compileflag:compileflagoverride //override:override_test",
)

# Test that fields are added to the config
plugin_e2e_test(
    name = "init_plugin_test",
    expect_output_contains = {
        ".plzconfig": "[Plugin \"python\"]*Target = //plugins:python",
    },
    # Need to break hard link here so init_plugin_test's actual .plzconfig doesn't get updated.
    plz_command = "cp .plzconfig .foo; mv .foo .plzconfig; plz init plugin python",
)

# Test that the target is created in plugins/BUILD
plugin_e2e_test(
    name = "init_create_target_test",
    expect_output_contains = {
        "plugins/BUILD": "plugin_repo*name = \\\"cc\\\"*plugin = \\\"cc-rules\\\"",
    },
    # Need to break hard link here so init_plugin_test's actual .plzconfig doesn't get updated.
    plz_command = "cp .plzconfig .foo; mv .foo .plzconfig; plz init plugin cc",
)

# Test that we don't add the build target again if it's already there
plugin_e2e_test(
    name = "init_only_add_target_once_test",
    # Need to break hard link here so init_plugin_test's actual .plzconfig doesn't get updated.
    plz_command = "cp .plzconfig .foo; mv .foo .plzconfig; plz init plugin cc; plz init plugin cc; lines=$(cat plugins/BUILD | wc -l); if [ $lines -gt 6 ]; then exit 1; else exit 0; fi",
)

plugin_e2e_test(
    name = "inherit_default_test",
    expected_output = {
        "plz-out/gen/foo/test/bar_inherit_test": "bar",
    },
    plz_command = "plz build -o plugin.foo.modulepath:something ///foo//test:bar_inherit_test",
)

please_repo_e2e_test(
    name = "override_default_in_plugin_test",
    expected_output = {
        "plz-out/gen/test/bar_inherit_test": "bar_test_tool",
    },
    plz_command = "plz build -o plugin.foo.modulepath:something //test:bar_inherit_test",
    repo = "foo_plugin",
)