subinclude("//test/export:export_e2e_test_build_def")

# Export a target from a repo that preloads a build def.
# This test purposely doesn't use the custom def but checks that the source files are still included.
please_export_e2e_test(
    name = "export_preload_genrule",
    export_targets = ["//:native_target"],
)
