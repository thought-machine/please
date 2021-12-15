subinclude("//test/build_defs:build_defs")


please_repo_e2e_test(
    name = "plugin_test",
    plz_command = "plz build -o foo.modulepath:something //test:foo_lib",
    repo = "test_repo",
)

please_repo_e2e_test(
    name = "nested_test",
    plz_command = "plz build -o foo.modulepath:something ///nested//test:test_lib ///foo//test:test_lib",
    expected_output = {
        "plz-out/gen/foo/test/test_lib.foo": "sha1 88f20b91c6af94a7974d7b547ec49529138a98f2 foolang/test.foo",
        "plz-out/gen/nested/test/test_lib.foo": "sha256 3e9e58ebd72c21509355abe528401ae4b37f632dc9446de9c1896235fa158856 nested/test.foo",
    },
    repo = "test_repo",
)

please_repo_e2e_test(
    name = "plugin_test_non_optional_config",
    expected_failure = True,
    plz_command = "plz build //test:foo_lib",
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
