subinclude("//test/export:export_e2e_test_build_def")

# Test --notrim flag.
please_export_e2e_test(
    name = "export_notrim",
    enforce_different_repos = False,
    export_targets = ["//:target1"],
    flags = "--notrim",
)
