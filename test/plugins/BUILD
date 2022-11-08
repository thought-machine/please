subinclude("//test/build_defs:build_defs")

please_repo_e2e_test(
    name = "plugin_test",
    plz_command = "plz build -o plugin.foo.modulepath:something //test:root_lib",
    repo = "test_repo",
)

please_repo_e2e_test(
    name = "cross_compile_test",
    plz_command = "plz build -o plugin.foo.modulepath:something ///linux_arm64//test:root_lib",
    repo = "test_repo",
)

please_repo_e2e_test(
    name = "nested_test",
    expected_output = {
        "plz-out/gen/foo/test/foo_lib.foo": "sha1 88f20b91c6af94a7974d7b547ec49529138a98f2 foolang/test.foo",
        "plz-out/gen/nested/test/nested_lib.foo": "sha256 3e9e58ebd72c21509355abe528401ae4b37f632dc9446de9c1896235fa158856 nested/test.foo",
    },
    plz_command = "plz build -o plugin.foo.modulepath:something ///nested//test:nested_lib ///foo//test:foo_lib",
    repo = "test_repo",
)

please_repo_e2e_test(
    name = "plugin_test_non_optional_config",
    expected_failure = True,
    plz_command = "plz build //test:root_lib",
    repo = "test_repo",
)

please_repo_e2e_test(
    name = "override_test",
    expected_output = {
        "plz-out/gen/override/fooc": "tool: overriden, path: something, flags: [compileflagoverride]",
    },
    plz_command = "plz build -o plugin.foo.modulepath:something -o plugin.foo.compileflag:compileflagoverride //override:override_test",
    repo = "test_repo",
)

# Test that fields are added to the config
please_repo_e2e_test(
    name = "init_plugin_test",
    expect_output_contains = {
        ".plzconfig": "[Plugin \"python\"]*Target = //plugins:python",
    },
    # Need to break hard link here so init_plugin_test's actual .plzconfig doesn't get updated.
    plz_command = "cp .plzconfig .foo; mv .foo .plzconfig; plz init plugin python",
    repo = "init_plugin_test",
)

# Test that the target is created in plugins/BUILD
please_repo_e2e_test(
    name = "init_create_target_test",
    expect_output_contains = {
        "plugins/BUILD": "plugin_repo*name = \\\"cc\\\"*plugin = \\\"cc-rules\\\"",
    },
    # Need to break hard link here so init_plugin_test's actual .plzconfig doesn't get updated.
    plz_command = "cp .plzconfig .foo; mv .foo .plzconfig; plz init plugin cc",
    repo = "init_plugin_test",
)

# Test that we don't add the build target again if it's already there
please_repo_e2e_test(
    name = "init_only_add_target_once_test",
    # Need to break hard link here so init_plugin_test's actual .plzconfig doesn't get updated.
    plz_command = "cp .plzconfig .foo; mv .foo .plzconfig; plz init plugin cc; plz init plugin cc; lines=$(cat plugins/BUILD | wc -l); if [ $lines -gt 6 ]; then exit 1; else exit 0; fi",
    repo = "init_plugin_test",
)

please_repo_e2e_test(
    name = "inherit_default_test",
    expected_output = {
        "plz-out/gen/foo/test/bar_inherit_test": "bar",
        "plugins/foo_plugin/plz-out/gen/test/bar_inherit_test": "overridden",
    },
    plz_command = "plz build -o plugin.foo.modulepath:something ///foo//test:bar_inherit_test && (cd plugins/foo_plugin && plz build  -o plugin.foo.modulepath:something //test:bar_inherit_test)",
    repo = "test_repo",
)
