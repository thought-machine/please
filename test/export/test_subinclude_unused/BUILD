subinclude("//test/export:export_e2e_test_build_def")

# Verifies that when a build definition statically subincludes another target but does not
# actively invoke its methods, the subincluded target is still fully exported to ensure
# static subinclude parsing succeeds in the exported repository.
please_export_e2e_test(
    name = "subinclude_unused_export_test",
    export_targets = ["//:file"],
)
