subinclude("//test/export:export_e2e_test_build_def")

# Export a custom target from a repo that preloads the build def used.
please_export_e2e_test(
    name = "export_preload_use_preloaded_def",
    export_targets = ["//:use_preloaded_def"],
)
