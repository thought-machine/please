subinclude("//test/build_defs")

please_repo_e2e_test(
    name = "custom_arch_test",
    expected_output = {
        "plz-out/gen/darwin_ppc/test/arch": "darwin_ppc",
    },
    plz_command = "plz build ///darwin_ppc//test:print_arch",
    repo = "test_repo",
)
