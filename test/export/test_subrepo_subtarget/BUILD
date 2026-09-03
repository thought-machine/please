subinclude("//test/export:export_e2e_test_build_def")

# Verifies that `plz export` correctly performs lookups to resolve internally generated sub-targets
# belonging to packages defined inside subrepos.
please_export_e2e_test(
    name = "subrepo_subtarget_export_test",
    export_targets = ["//:target"],
)
