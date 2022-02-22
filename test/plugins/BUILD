subinclude("//test/build_defs:build_defs")

please_repo_e2e_test(
    name = "plugin_test",
    plz_command = "plz build -o foo.modulepath:something //test:root_lib",
    repo = "test_repo",
)

please_repo_e2e_test(
    name = "cross_compile_test",
    plz_command = "plz build -o foo.modulepath:something ///linux_arm64//test:root_lib",
    repo = "test_repo",
)

please_repo_e2e_test(
    name = "nested_test",
    expected_output = {
        "plz-out/gen/foo/test/foo_lib.foo": "sha1 88f20b91c6af94a7974d7b547ec49529138a98f2 foolang/test.foo",
        "plz-out/gen/nested/test/nested_lib.foo": "sha256 3e9e58ebd72c21509355abe528401ae4b37f632dc9446de9c1896235fa158856 nested/test.foo",
    },
    plz_command = "plz build -o foo.modulepath:something ///nested//test:nested_lib ///foo//test:foo_lib",
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
        "plz-out/gen/override/fooc": "overriden",
    },
    plz_command = "plz build -o foo.modulepath:something //override:override_test",
    repo = "test_repo",
)

please_repo_e2e_test(
    name = "init_plugin_test",
    expect_output_contains = {
        ".plzconfig" : "[Plugin \"python\"]",
        ".plzconfig" : "; DefaultInterpreter = python3",
    },
    plz_command = "> .plzconfig && plz init plugin python",
    repo = "init_plugin_test",
)

please_repo_e2e_test(
    name = "init_create_target_test",
    expect_output_contains = {
        "plugins/BUILD" : "plugin_repo",
        "plugins/BUILD" : "name = \\\"cc\\\"",
        "plugins/BUILD" : "plugin = \\\"cc-rules\\\"",
    },
    plz_command = "plz init plugin cc",
    repo = "init_plugin_test",
)
