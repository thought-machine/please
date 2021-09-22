subinclude("//test/build_defs:build_defs")

please_repo_e2e_test(
    name = "plugin_test",
    plz_command = "plz build -o foo.modulepath:something //test:foo_lib",
    repo = "test_repo",
)

please_repo_e2e_test(
    name = "plugin_test_non_optional_config",
    plz_command = "plz build //test:foo_lib",
    repo = "test_repo",
    expected_failure = True,
)