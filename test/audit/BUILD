subinclude("//test/build_defs")

please_repo_e2e_test(
    name = "audit_test",
    repo = "test_repo",
    plz_command = "./test_audit.sh",
    # Asserts that the plugin download is audited, which only happens if we actually download it.
    shared_cache = False,
)
