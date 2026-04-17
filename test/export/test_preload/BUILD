subinclude("//test/export:export_e2e_test_build_def")

# Export a target from a repo that preloads a build def.
# The exported target doesn't use the preloaded build def, but the preloaded build def is still
# present in the export.
please_export_e2e_test(
    name = "export_preload_genrule",
    export_targets = ["//:native_target"],
)
