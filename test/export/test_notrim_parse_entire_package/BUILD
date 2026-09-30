subinclude("//test/export:export_e2e_test_build_def")

# Test --notrim exports adjacent targets and dependencies from packages that would not have been
# included in the original parse. This is a regression test to make sure we enforce entire package
# parsing when using the export command.
please_export_e2e_test(
    name = "export_notrim_parse_entire_package",
    enforce_different_repos = False,
    export_targets = ["//:target1"],
    flags = "--notrim",
)
