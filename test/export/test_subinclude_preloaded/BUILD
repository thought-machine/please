subinclude("//test/export:export_e2e_test_build_def")

# Verifies that `plz export` correctly processes a subincluded build definition target
# that in turn subincludes a preloaded target.
please_export_e2e_test(
    name = "subinclude_preloaded_export_test",
    export_targets = ["//:file"],
)