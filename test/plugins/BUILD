subinclude("//test/build_defs:build_defs")

genrule(
    name = "foo_plugin",
    srcs = {
        "plugin": ["foo_plugin"],
        "tool": ["//test/plugins/tools:fooc"],
    },
    outs = ["foo_plugin.tar.gz"],
    cmd = "mv $SRCS_PLUGIN foo_plugin && mv $SRCS_TOOL foo_plugin/tools && tar -czf $OUT foo_plugin",
)

def plugin_e2e_test(name, plz_command, expected_output=None, expected_failure=False, expect_output_contains=None):
    return please_repo_e2e_test(
        name = name,
        data = {
            "plugin": [":foo_plugin"],
            "nested_subrepo": ["nested_subrepo"],
        },
        expect_output_contains = expect_output_contains,
        expected_failure = expected_failure,
        expected_output = expected_output,
        plz_command = plz_command.replace("plz ", "plz -o buildconfig.nested-subrepo:$TMP_DIR/$DATA_NESTED_SUBREPO -o please.pluginrepo:file://$TMP_DIR/$DATA_PLUGIN "),
        repo = "test_repo",
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
        "plz-out/gen/override/fooc": "fooc: overriden, bar: bar, path: something, flags: [compileflagoverride]",
    },
    plz_command = "plz build -o plugin.foo.modulepath:something -o plugin.foo.compileflag:compileflagoverride //override:override_test",
)

plugin_e2e_test(
    name = "profile_test",
    expected_output = {
        "plz-out/gen/override/fooc": "fooc: overriden, bar: bar-dev, path: something, flags: []",
    },
    plz_command = "plz build -o plugin.foo.modulepath:something --profile dev --profile e2e //override:override_test",
)

# Test that fields are added to the config
plugin_e2e_test(
    name = "init_plugin_test",
    expect_output_contains = {
        ".plzconfig": "[Plugin \"python\"]*Target = //plugins:python",
    },
    # Need to break hard link here so init_plugin_test's actual .plzconfig doesn't get updated.
    plz_command = "cp .plzconfig .foo; mv .foo .plzconfig; plz init plugin python --version v1.2.0",
)

plugin_e2e_test(
    name = "init_go_test",
    expect_output_contains = {
        "go.conf": "//third_party/go:toolchain|go\n//third_party/go:std",
    },
    # Need to break hard link here so init_plugin_test's actual .plzconfig doesn't get updated.
    plz_command = "cp .plzconfig .foo; mv .foo .plzconfig; plz init plugin go && plz query config plugin.go.gotool > go.conf && plz query config plugin.go.stdlib >> go.conf",
)

# Test that the target is created in plugins/BUILD
plugin_e2e_test(
    name = "init_create_target_test",
    expect_output_contains = {
        "plugins/BUILD": "plugin_repo*name = \\\"cc\\\"*plugin = \\\"cc-rules\\\"",
    },
    # Need to break hard link here so init_plugin_test's actual .plzconfig doesn't get updated.
    plz_command = "cp .plzconfig .foo; mv .foo .plzconfig; plz init plugin cc --version v0.3.2",
)

# Test that we don't add the build target again if it's already there
plugin_e2e_test(
    name = "init_only_add_target_once_test",
    # Need to break hard link here so init_plugin_test's actual .plzconfig doesn't get updated.
    plz_command = "cp .plzconfig .foo; mv .foo .plzconfig; plz init plugin cc --version v0.3.2; plz init plugin cc --version v0.3.2; lines=$(cat plugins/BUILD | wc -l); if [ $lines -gt 6 ]; then exit 1; else exit 0; fi",
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

# ///foo//build_defs:foo references itself by its plugin name which didn't work when the plugin was also the host repo
please_repo_e2e_test(
    name = "self_referencing_plugin_test",
    expected_output = {
        "plz-out/gen/test/test.txt": "foo",
    },
    plz_command = "plz build  -o plugin.foo.modulepath:something //test/...",
    repo = "self_referencing_plugin",
)

plugin_e2e_test(
    name = "arch_test",
    expected_output = {
        "plz-out/gen/foo_bar66/fooc.txt": "bar66",
    },
    plz_command = "plz build -o plugin.foo.modulepath:something --arch foo_bar66 //:output_fooc",
)
