subinclude("//test/build_defs")


please_repo_e2e_test(
    name = "arch_subrepo_config",
    expected_failure = False,
    plz_command = "plz build --arch test_arch //:testFile",
    repo = "test_repo"
)